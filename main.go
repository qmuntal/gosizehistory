package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	goversion "go/version"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/qmuntal/gosizehistory/internal/goreleases"
	"github.com/qmuntal/gosizehistory/internal/history"
	"github.com/qmuntal/gosizehistory/internal/tipbuild"
)

type options struct {
	metadataSource  string
	downloadBase    string
	output          string
	cacheDir        string
	version         string
	goos            string
	goarch          string
	latestPerMinor  bool
	refreshStable   bool
	tip             bool
	tipRepository   string
	tipRef          string
	tipBase         string
	mergeTipReport  string
	bootstrapGOROOT string
	tipWorkers      int
	workers         int
	retries         int
	dryRun          bool
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	if err == nil || errors.Is(err, flag.ErrHelp) {
		return
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	var opts options
	flags := flag.NewFlagSet("gosizehistory", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.metadataSource, "metadata", goreleases.DefaultMetadataURL, "Go release metadata URL or local JSON file")
	flags.StringVar(&opts.downloadBase, "download-base-url", history.DefaultDownloadBaseURL, "base URL containing Go archives")
	flags.StringVar(&opts.output, "output", "go-tool-sizes.json", "output JSON path, or - for stdout")
	flags.StringVar(&opts.cacheDir, "cache-dir", ".cache/gosizehistory", "directory for downloaded archives")
	flags.StringVar(&opts.version, "version", "", "exact Go version to include, such as go1.27.0")
	flags.StringVar(&opts.goos, "os", "", "exact GOOS to include, such as linux")
	flags.StringVar(&opts.goarch, "arch", "", "release architecture to include, such as amd64 or arm")
	flags.BoolVar(&opts.latestPerMinor, "latest-per-minor", false, "include only the latest stable patch of each Go minor version")
	flags.BoolVar(&opts.refreshStable, "refresh-stable", false, "merge only new latest stable patch releases into the existing output report")
	flags.BoolVar(&opts.tip, "tip", false, "build a report from golang/go tip instead of release archives")
	flags.StringVar(&opts.tipRepository, "tip-repository", tipbuild.DefaultRepository, "Git repository used for tip builds")
	flags.StringVar(&opts.tipRef, "tip-ref", "HEAD", "Git ref fetched for the tip build")
	flags.StringVar(&opts.tipBase, "tip-base", "", "base report merged with tip (defaults to output path)")
	flags.StringVar(&opts.mergeTipReport, "merge-tip-report", "", "merge an existing tip-only report without rebuilding")
	flags.StringVar(&opts.bootstrapGOROOT, "bootstrap-goroot", "", "bootstrap GOROOT for building Go tip (defaults to this binary's GOROOT)")
	flags.IntVar(&opts.tipWorkers, "tip-workers", 1, "number of target toolchains to build concurrently")
	flags.IntVar(&opts.workers, "workers", min(runtime.NumCPU(), 4), "number of archives to process concurrently")
	flags.IntVar(&opts.retries, "retries", 3, "download attempts per archive")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "show the selected download size without downloading")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	if opts.workers < 1 {
		return fmt.Errorf("workers must be at least 1")
	}
	if opts.retries < 1 {
		return fmt.Errorf("retries must be at least 1")
	}
	if opts.tipWorkers < 1 {
		return fmt.Errorf("tip-workers must be at least 1")
	}
	if opts.output == "" {
		return fmt.Errorf("output path must not be empty")
	}
	if opts.latestPerMinor && opts.version != "" {
		return fmt.Errorf("latest-per-minor cannot be combined with version")
	}
	if opts.refreshStable && (opts.latestPerMinor || opts.version != "" || opts.goos != "" || opts.goarch != "") {
		return fmt.Errorf("refresh-stable cannot be combined with latest-per-minor, version, os, or arch")
	}
	if (opts.tip || opts.mergeTipReport != "") && (opts.latestPerMinor || opts.version != "") {
		return fmt.Errorf("tip cannot be combined with latest-per-minor or version")
	}
	if opts.refreshStable && (opts.tip || opts.mergeTipReport != "") {
		return fmt.Errorf("refresh-stable cannot be combined with tip or merge-tip-report")
	}
	if (opts.tip || opts.mergeTipReport != "") && opts.dryRun {
		return fmt.Errorf("dry-run is not supported with tip")
	}
	if opts.tip && opts.mergeTipReport != "" {
		return fmt.Errorf("tip cannot be combined with merge-tip-report")
	}
	if opts.refreshStable && opts.output == "-" {
		return fmt.Errorf("refresh-stable requires an output file")
	}

	logger := log.New(stderr, "", 0)
	if opts.mergeTipReport != "" {
		tipReport, err := history.ReadFile(opts.mergeTipReport)
		if err != nil {
			return fmt.Errorf("read tip report: %w", err)
		}
		return mergeAndWriteReport(opts.output, opts.tipBase, tipReport, stdout, logger)
	}
	if opts.tip {
		tipReport, err := tipbuild.Build(ctx, tipbuild.Config{
			WorkDir:         filepath.Join(opts.cacheDir, "tip"),
			Repository:      opts.tipRepository,
			Ref:             opts.tipRef,
			BootstrapGOROOT: opts.bootstrapGOROOT,
			GOOS:            opts.goos,
			GOARCH:          opts.goarch,
			Workers:         opts.tipWorkers,
			Status: func(message string) {
				logger.Print(message)
			},
			Progress: func(progress tipbuild.Progress) {
				logger.Printf("[%d/%d] %s/%s (built)", progress.Completed, progress.Total, progress.Target.OS, progress.Target.Arch)
			},
		})
		if err != nil {
			return err
		}
		return mergeAndWriteReport(opts.output, opts.tipBase, tipReport, stdout, logger)
	}

	var stableBase history.Report
	if opts.refreshStable {
		var err error
		stableBase, err = history.ReadFile(opts.output)
		if err != nil {
			return fmt.Errorf("read stable base report %s: %w", opts.output, err)
		}
	}

	client := newHTTPClient(opts.workers)
	plan, err := history.NewPlan(ctx, client, opts.metadataSource, goreleases.Filter{
		Version:        opts.version,
		OS:             opts.goos,
		Arch:           opts.goarch,
		LatestPerMinor: opts.latestPerMinor || opts.refreshStable,
	})
	if err != nil {
		return err
	}
	if opts.refreshStable {
		plan = newStablePlan(plan, stableBase)
		if len(plan.Archives) == 0 {
			logger.Print("no new stable releases")
			return nil
		}
	}

	knownSize, unknownSizes := plan.KnownDownloadSize()
	if unknownSizes == 0 {
		logger.Printf("selected %d archives across %d releases (%s)", len(plan.Archives), plan.ReleaseCount(), formatBytes(knownSize))
	} else {
		logger.Printf("selected %d archives across %d releases (%s plus %d archives of unknown size)", len(plan.Archives), plan.ReleaseCount(), formatBytes(knownSize), unknownSizes)
	}
	if opts.dryRun {
		return nil
	}

	report, err := history.Build(ctx, plan, history.BuildConfig{
		CacheDir: opts.cacheDir,
		BaseURL:  opts.downloadBase,
		Client:   client,
		Workers:  opts.workers,
		Retries:  opts.retries,
		Progress: func(progress history.Progress) {
			state := "downloaded"
			if progress.Cached {
				state = "cached"
			}
			logger.Printf("[%d/%d] %s (%s)", progress.Completed, progress.Total, progress.Filename, state)
		},
	})
	if err != nil {
		return err
	}
	if opts.refreshStable {
		report, err = history.MergeStable(stableBase, report)
		if err != nil {
			return err
		}
		return writeReport(opts.output, report, stdout, logger)
	}

	if opts.output != "-" {
		if existing, readErr := history.ReadFile(opts.output); readErr == nil {
			report, err = history.PreserveDevelopment(report, existing)
			if err != nil {
				return err
			}
		} else if !errors.Is(readErr, os.ErrNotExist) {
			return fmt.Errorf("read existing output report: %w", readErr)
		}
	}
	return writeReport(opts.output, report, stdout, logger)
}

func newStablePlan(plan history.Plan, existing history.Report) history.Plan {
	existingVersions := make(map[string]string)
	for _, release := range existing.Releases {
		if release.Stable {
			minor := goversion.Lang(release.Version)
			if current := existingVersions[minor]; current == "" || goversion.Compare(release.Version, current) > 0 {
				existingVersions[minor] = release.Version
			}
		}
	}
	archives := make([]goreleases.Archive, 0, len(plan.Archives))
	for _, archive := range plan.Archives {
		current := existingVersions[goversion.Lang(archive.Release)]
		if current == "" || goversion.Compare(archive.Release, current) > 0 {
			archives = append(archives, archive)
		}
	}
	plan.Archives = archives
	return plan
}

func mergeAndWriteReport(output, basePath string, tipReport history.Report, stdout io.Writer, logger *log.Logger) error {
	if basePath == "" {
		if output == "-" {
			return fmt.Errorf("tip-base is required when output is stdout")
		}
		basePath = output
	}
	base, err := history.ReadFile(basePath)
	if err != nil {
		return fmt.Errorf("read tip base report %s: %w", basePath, err)
	}
	merged, err := history.MergeDevelopment(base, tipReport)
	if err != nil {
		return err
	}
	return writeReport(output, merged, stdout, logger)
}

func writeReport(output string, report history.Report, stdout io.Writer, logger *log.Logger) error {
	if output == "-" {
		return history.Write(stdout, report)
	}
	if err := history.WriteFile(output, report); err != nil {
		return err
	}
	logger.Printf("wrote %s", output)
	return nil
}

func newHTTPClient(workers int) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = workers
	transport.MaxConnsPerHost = workers
	return &http.Client{Transport: transport}
}

func formatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	for _, unit := range units {
		value /= 1024
		if value < 1024 || unit == units[len(units)-1] {
			return fmt.Sprintf("%.2f %s", value, unit)
		}
	}
	return fmt.Sprintf("%d B", size)
}
