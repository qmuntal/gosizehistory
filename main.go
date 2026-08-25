package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"

	"github.com/qmuntal/gosizehistory/internal/goreleases"
	"github.com/qmuntal/gosizehistory/internal/history"
)

type options struct {
	metadataSource string
	downloadBase   string
	output         string
	cacheDir       string
	version        string
	goos           string
	goarch         string
	latestPerMinor bool
	workers        int
	retries        int
	dryRun         bool
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
	flags.StringVar(&opts.goarch, "arch", "", "exact release architecture to include, such as amd64")
	flags.BoolVar(&opts.latestPerMinor, "latest-per-minor", false, "include only the latest stable patch of each Go minor version")
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
	if opts.output == "" {
		return fmt.Errorf("output path must not be empty")
	}
	if opts.latestPerMinor && opts.version != "" {
		return fmt.Errorf("latest-per-minor cannot be combined with version")
	}

	client := newHTTPClient(opts.workers)
	plan, err := history.NewPlan(ctx, client, opts.metadataSource, goreleases.Filter{
		Version:        opts.version,
		OS:             opts.goos,
		Arch:           opts.goarch,
		LatestPerMinor: opts.latestPerMinor,
	})
	if err != nil {
		return err
	}

	logger := log.New(stderr, "", 0)
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

	if opts.output == "-" {
		return history.Write(stdout, report)
	}
	if err := history.WriteFile(opts.output, report); err != nil {
		return err
	}
	logger.Printf("wrote %s", opts.output)
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
