package tipbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/qmuntal/gosizehistory/internal/history"
	"github.com/qmuntal/gosizehistory/internal/platform"
)

const DefaultRepository = "https://github.com/golang/go.git"

const releaseGOFLAGS = "-trimpath -ldflags=-w -gcflags=cmd/...=-dwarf=false"

var developmentVersionPattern = regexp.MustCompile(`\bgo(\d+\.\d+)`)

type Config struct {
	WorkDir         string
	Repository      string
	Ref             string
	BootstrapGOROOT string
	GOOS            string
	GOARCH          string
	Workers         int
	Now             func() time.Time
	Status          func(string)
	Progress        func(Progress)
}

type Progress struct {
	Completed int
	Total     int
	Target    Target
}

func Build(ctx context.Context, config Config) (history.Report, error) {
	if config.WorkDir == "" {
		return history.Report{}, fmt.Errorf("tip work directory must not be empty")
	}
	workDir, err := filepath.Abs(config.WorkDir)
	if err != nil {
		return history.Report{}, fmt.Errorf("resolve tip work directory: %w", err)
	}
	config.WorkDir = workDir
	if config.Repository == "" {
		config.Repository = DefaultRepository
	}
	if config.Ref == "" {
		config.Ref = "HEAD"
	}
	if config.BootstrapGOROOT == "" {
		config.BootstrapGOROOT = runtime.GOROOT()
	}
	if config.Workers < 1 {
		return history.Report{}, fmt.Errorf("tip workers must be at least 1")
	}

	sourceDir := filepath.Join(config.WorkDir, "source")
	if err := prepareCheckout(ctx, config, sourceDir); err != nil {
		return history.Report{}, err
	}
	if config.Status != nil {
		config.Status("bootstrapping Go tip for the host")
	}
	if err := bootstrap(ctx, sourceDir, config.BootstrapGOROOT); err != nil {
		return history.Report{}, err
	}

	goCommand := filepath.Join(sourceDir, "bin", "go"+executableSuffix(runtime.GOOS))
	targets, err := listTargets(ctx, goCommand, sourceDir, config.GOOS, config.GOARCH)
	if err != nil {
		return history.Report{}, err
	}
	if len(targets) == 0 {
		return history.Report{}, fmt.Errorf("no standalone tip toolchain targets match the selected filters")
	}

	metadata, err := readMetadata(ctx, goCommand, sourceDir)
	if err != nil {
		return history.Report{}, err
	}
	if config.Status != nil {
		config.Status(fmt.Sprintf("building Go tip %s for %d platforms", metadata.Revision[:12], len(targets)))
	}

	outputDir := filepath.Join(config.WorkDir, "output")
	if err := os.RemoveAll(outputDir); err != nil {
		return history.Report{}, fmt.Errorf("clean tip output: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return history.Report{}, fmt.Errorf("create tip output: %w", err)
	}
	gocache := filepath.Join(config.WorkDir, "gocache")
	if err := os.RemoveAll(gocache); err != nil {
		return history.Report{}, fmt.Errorf("clean tip build cache: %w", err)
	}

	platforms, err := buildTargets(ctx, goCommand, sourceDir, outputDir, gocache, targets, config)
	if err != nil {
		return history.Report{}, err
	}
	now := time.Now
	if config.Now != nil {
		now = config.Now
	}
	commitTime := metadata.CommitTime.UTC()
	return history.Report{
		SchemaVersion: history.SchemaVersion,
		GeneratedAt:   now().UTC(),
		Source:        config.Repository + "@" + metadata.Revision,
		Releases: []history.Release{{
			Version:     metadata.Version,
			Development: true,
			Source:      config.Repository,
			Revision:    metadata.Revision,
			CommitTime:  &commitTime,
			Platforms:   platforms,
		}},
	}, nil
}

type metadata struct {
	Version    string
	Revision   string
	CommitTime time.Time
}

func prepareCheckout(ctx context.Context, config Config, sourceDir string) error {
	if _, err := os.Stat(filepath.Join(sourceDir, ".git")); os.IsNotExist(err) {
		if err := os.RemoveAll(sourceDir); err != nil {
			return fmt.Errorf("clean incomplete tip checkout: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(sourceDir), 0o755); err != nil {
			return fmt.Errorf("create tip work directory: %w", err)
		}
		if config.Status != nil {
			config.Status("cloning golang/go")
		}
		if _, err := run(ctx, "", nil, "git", "clone", "--filter=blob:none", "--no-checkout", config.Repository, sourceDir); err != nil {
			return fmt.Errorf("clone Go repository: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("inspect tip checkout: %w", err)
	}

	if config.Status != nil {
		config.Status("fetching golang/go origin " + config.Ref)
	}
	if _, err := run(ctx, sourceDir, nil, "git", "remote", "set-url", "origin", config.Repository); err != nil {
		return fmt.Errorf("configure Go origin: %w", err)
	}
	if _, err := run(ctx, sourceDir, nil, "git", "fetch", "--depth=1", "origin", config.Ref); err != nil {
		return fmt.Errorf("fetch Go origin %s: %w", config.Ref, err)
	}
	if _, err := run(ctx, sourceDir, nil, "git", "checkout", "--detach", "--force", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("checkout Go origin %s: %w", config.Ref, err)
	}
	if _, err := run(ctx, sourceDir, nil, "git", "clean", "-ffdx"); err != nil {
		return fmt.Errorf("clean Go checkout: %w", err)
	}
	return nil
}

func bootstrap(ctx context.Context, sourceDir, bootstrapGOROOT string) error {
	environment := cleanEnvironment(map[string]string{
		"CGO_ENABLED":      "0",
		"GOENV":            "off",
		"GOFLAGS":          "",
		"GOROOT_BOOTSTRAP": bootstrapGOROOT,
		"GOTOOLCHAIN":      "local",
	}, "GOOS", "GOARCH", "GOHOSTOS", "GOHOSTARCH", "GOBIN", "GOROOT")

	var name string
	var args []string
	if runtime.GOOS == "windows" {
		name = "cmd.exe"
		args = []string{"/d", "/c", "make.bat"}
	} else {
		name = "bash"
		args = []string{"make.bash"}
	}
	if _, err := run(ctx, filepath.Join(sourceDir, "src"), environment, name, args...); err != nil {
		return fmt.Errorf("bootstrap Go tip: %w", err)
	}
	return nil
}

func listTargets(ctx context.Context, goCommand, sourceDir, goos, goarch string) ([]Target, error) {
	output, err := run(ctx, sourceDir, goEnvironment(sourceDir, "", "", ""), goCommand, "tool", "dist", "list", "-json")
	if err != nil {
		return nil, fmt.Errorf("list Go tip targets: %w", err)
	}
	var targets []Target
	if err := json.Unmarshal(output, &targets); err != nil {
		return nil, fmt.Errorf("decode Go tip targets: %w", err)
	}
	return DistributionTargets(targets, goos, goarch), nil
}

func readMetadata(ctx context.Context, goCommand, sourceDir string) (metadata, error) {
	revisionBytes, err := run(ctx, sourceDir, nil, "git", "rev-parse", "HEAD")
	if err != nil {
		return metadata{}, fmt.Errorf("read Go tip revision: %w", err)
	}
	revision := strings.TrimSpace(string(revisionBytes))
	if len(revision) < 12 {
		return metadata{}, fmt.Errorf("read Go tip revision: invalid revision %q", revision)
	}
	timeBytes, err := run(ctx, sourceDir, nil, "git", "show", "-s", "--format=%cI", "HEAD")
	if err != nil {
		return metadata{}, fmt.Errorf("read Go tip commit time: %w", err)
	}
	commitTime, err := time.Parse(time.RFC3339, strings.TrimSpace(string(timeBytes)))
	if err != nil {
		return metadata{}, fmt.Errorf("parse Go tip commit time: %w", err)
	}
	versionBytes, err := run(ctx, sourceDir, goEnvironment(sourceDir, "", "", ""), goCommand, "env", "GOVERSION")
	if err != nil {
		return metadata{}, fmt.Errorf("read Go tip version: %w", err)
	}
	version, err := normalizeDevelopmentVersion(string(versionBytes))
	if err != nil {
		return metadata{}, err
	}
	return metadata{Version: version, Revision: revision, CommitTime: commitTime}, nil
}

func normalizeDevelopmentVersion(version string) (string, error) {
	match := developmentVersionPattern.FindStringSubmatch(version)
	if len(match) != 2 {
		return "", fmt.Errorf("read Go tip version: cannot normalize %q", strings.TrimSpace(version))
	}
	return "go" + match[1] + "-tip", nil
}

func buildTargets(ctx context.Context, goCommand, sourceDir, outputDir, gocache string, targets []Target, config Config) ([]history.Platform, error) {
	buildContext, cancel := context.WithCancel(ctx)
	defer cancel()

	platforms := make([]history.Platform, len(targets))
	jobs := make(chan int)
	var workers sync.WaitGroup
	var firstErr error
	var firstErrOnce sync.Once
	var completed atomic.Int64
	for range config.Workers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-buildContext.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					target := targets[index]
					platform, err := buildTarget(buildContext, goCommand, sourceDir, outputDir, gocache, target)
					if err != nil {
						firstErrOnce.Do(func() {
							firstErr = err
							cancel()
						})
						return
					}
					platforms[index] = platform
					finished := int(completed.Add(1))
					if config.Progress != nil {
						config.Progress(Progress{Completed: finished, Total: len(targets), Target: target})
					}
				}
			}
		}()
	}

sendJobs:
	for index := range targets {
		select {
		case jobs <- index:
		case <-buildContext.Done():
			break sendJobs
		}
	}
	close(jobs)
	workers.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return platforms, nil
}

func buildTarget(ctx context.Context, goCommand, sourceDir, outputDir, gocache string, target Target) (history.Platform, error) {
	targetDir := filepath.Join(outputDir, target.OS+"_"+target.Arch)
	targetCache := filepath.Join(gocache, target.OS+"_"+target.Arch)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return history.Platform{}, fmt.Errorf("build tip %s/%s: %w", target.OS, target.Arch, err)
	}
	defer os.RemoveAll(targetDir)
	defer os.RemoveAll(targetCache)
	args := []string{"build", "-a", "-buildvcs=false", "-o", targetDir}
	args = append(args, distributionPackages...)
	if _, err := run(ctx, filepath.Join(sourceDir, "src"), targetEnvironment(sourceDir, targetCache, target), goCommand, args...); err != nil {
		return history.Platform{}, fmt.Errorf("build tip %s/%s: %w", target.OS, target.Arch, err)
	}
	tools, err := Inventory(targetDir, target)
	if err != nil {
		return history.Platform{}, err
	}
	return history.Platform{OS: target.OS, Arch: reportArch(target), Tools: tools}, nil
}

func targetEnvironment(goroot, gocache string, target Target) []string {
	environment := goEnvironment(goroot, target.OS, target.Arch, gocache)
	environment = setEnvironment(environment, "GOFLAGS", releaseGOFLAGS)
	if target.OS == "linux" && target.Arch == "arm" {
		environment = setEnvironment(environment, "GOARM", "6")
	}
	return environment
}

func setEnvironment(environment []string, key, value string) []string {
	prefix := strings.ToUpper(key) + "="
	for index, entry := range environment {
		if strings.HasPrefix(strings.ToUpper(entry), prefix) {
			environment[index] = key + "=" + value
			sort.Strings(environment)
			return environment
		}
	}
	environment = append(environment, key+"="+value)
	sort.Strings(environment)
	return environment
}

func reportArch(target Target) string {
	return platform.CanonicalArch(target.Arch)
}

func goEnvironment(goroot, goos, goarch, gocache string) []string {
	overrides := map[string]string{
		"CGO_ENABLED": "0",
		"GOENV":       "off",
		"GOFLAGS":     "",
		"GOROOT":      goroot,
		"GOTOOLCHAIN": "local",
	}
	if goos != "" {
		overrides["GOOS"] = goos
	}
	if goarch != "" {
		overrides["GOARCH"] = goarch
	}
	if gocache != "" {
		overrides["GOCACHE"] = gocache
	}
	return cleanEnvironment(overrides,
		"GOBIN",
		"GO386",
		"GOAMD64",
		"GOARM",
		"GOEXPERIMENT",
		"GOMIPS",
		"GOMIPS64",
		"GOPPC64",
		"GORISCV64",
		"GOWASM",
	)
}

func cleanEnvironment(overrides map[string]string, unset ...string) []string {
	values := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[strings.ToUpper(key)] = key + "=" + value
		}
	}
	for _, key := range unset {
		delete(values, strings.ToUpper(key))
	}
	for key, value := range overrides {
		values[strings.ToUpper(key)] = key + "=" + value
	}
	environment := make([]string, 0, len(values))
	for _, entry := range values {
		environment = append(environment, entry)
	}
	sort.Strings(environment)
	return environment
}

func run(ctx context.Context, directory string, environment []string, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = directory
	if environment != nil {
		command.Env = environment
	}
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return nil, fmt.Errorf("%s: %w", message, err)
		}
		return nil, err
	}
	return output, nil
}

func executableSuffix(goos string) string {
	if goos == "windows" {
		return ".exe"
	}
	return ""
}
