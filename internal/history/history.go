package history

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/qmuntal/gosizehistory/internal/archivecache"
	"github.com/qmuntal/gosizehistory/internal/archiveinventory"
	"github.com/qmuntal/gosizehistory/internal/goreleases"
	"github.com/qmuntal/gosizehistory/internal/platform"
)

const (
	SchemaVersion          = 1
	DefaultDownloadBaseURL = "https://go.dev/dl/"
)

type Plan struct {
	Source   string
	Archives []goreleases.Archive
}

type BuildConfig struct {
	CacheDir string
	BaseURL  string
	Client   *http.Client
	Workers  int
	Retries  int
	Now      func() time.Time
	Progress func(Progress)
}

type Progress struct {
	Completed int
	Total     int
	Filename  string
	Cached    bool
}

type Report struct {
	SchemaVersion int       `json:"schema_version"`
	GeneratedAt   time.Time `json:"generated_at"`
	Source        string    `json:"source"`
	Releases      []Release `json:"releases"`
}

type Release struct {
	Version     string     `json:"version"`
	Stable      bool       `json:"stable"`
	Development bool       `json:"development,omitempty"`
	Source      string     `json:"source,omitempty"`
	Revision    string     `json:"revision,omitempty"`
	CommitTime  *time.Time `json:"commit_time,omitempty"`
	Platforms   []Platform `json:"platforms"`
}

type Platform struct {
	OS      string                  `json:"os"`
	Arch    string                  `json:"arch"`
	Archive *Archive                `json:"archive,omitempty"`
	Tools   []archiveinventory.Tool `json:"tools"`
}

type Archive struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
}

func NewPlan(ctx context.Context, client *http.Client, source string, filter goreleases.Filter) (Plan, error) {
	releases, err := goreleases.Load(ctx, client, source)
	if err != nil {
		return Plan{}, err
	}
	archives := goreleases.SelectArchives(releases, filter)
	if len(archives) == 0 {
		return Plan{}, fmt.Errorf("no release archives match the selected filters")
	}
	return Plan{Source: source, Archives: archives}, nil
}

func (plan Plan) KnownDownloadSize() (size int64, unknown int) {
	for _, archive := range plan.Archives {
		if archive.File.Size > 0 {
			size += archive.File.Size
		} else {
			unknown++
		}
	}
	return size, unknown
}

func (plan Plan) ReleaseCount() int {
	versions := make(map[string]struct{})
	for _, archive := range plan.Archives {
		versions[archive.Release] = struct{}{}
	}
	return len(versions)
}

func Build(ctx context.Context, plan Plan, config BuildConfig) (Report, error) {
	if config.Workers < 1 {
		return Report{}, fmt.Errorf("workers must be at least 1")
	}
	if config.CacheDir == "" {
		return Report{}, fmt.Errorf("cache directory must not be empty")
	}
	if config.BaseURL == "" {
		return Report{}, fmt.Errorf("download base URL must not be empty")
	}

	archiveCache := archivecache.Cache{
		Dir:     config.CacheDir,
		BaseURL: config.BaseURL,
		Client:  config.Client,
		Retries: config.Retries,
	}
	platforms := make([]Platform, len(plan.Archives))
	buildContext, cancel := context.WithCancel(ctx)
	defer cancel()

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
					selected := plan.Archives[index]
					cachedArchive, err := archiveCache.Get(buildContext, archivecache.Artifact{
						Filename: selected.File.Filename,
						Size:     selected.File.Size,
						SHA256:   selected.File.SHA256,
					})
					if err != nil {
						firstErrOnce.Do(func() {
							firstErr = fmt.Errorf("%s: %w", selected.File.Filename, err)
							cancel()
						})
						return
					}
					tools, err := archiveinventory.Inspect(cachedArchive.Path)
					if err != nil {
						firstErrOnce.Do(func() {
							firstErr = fmt.Errorf("inspect %s: %w", selected.File.Filename, err)
							cancel()
						})
						return
					}
					if tools == nil {
						tools = make([]archiveinventory.Tool, 0)
					}
					platforms[index] = Platform{
						OS:   selected.File.OS,
						Arch: platform.CanonicalArch(selected.File.Arch),
						Archive: &Archive{
							Filename: selected.File.Filename,
							Size:     cachedArchive.Size,
							SHA256:   cachedArchive.SHA256,
						},
						Tools: tools,
					}
					finished := int(completed.Add(1))
					if config.Progress != nil {
						config.Progress(Progress{
							Completed: finished,
							Total:     len(plan.Archives),
							Filename:  selected.File.Filename,
							Cached:    cachedArchive.Cached,
						})
					}
				}
			}
		}()
	}

sendJobs:
	for index := range plan.Archives {
		select {
		case jobs <- index:
		case <-buildContext.Done():
			break sendJobs
		}
	}
	close(jobs)
	workers.Wait()

	if firstErr != nil {
		return Report{}, firstErr
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}

	now := time.Now
	if config.Now != nil {
		now = config.Now
	}
	return Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   now().UTC(),
		Source:        plan.Source,
		Releases:      groupReleases(plan.Archives, platforms),
	}, nil
}

func groupReleases(archives []goreleases.Archive, platforms []Platform) []Release {
	releases := make([]Release, 0)
	positions := make(map[string]int)
	for index, selected := range archives {
		position, ok := positions[selected.Release]
		if !ok {
			position = len(releases)
			positions[selected.Release] = position
			releases = append(releases, Release{
				Version:   selected.Release,
				Stable:    selected.Stable,
				Platforms: make([]Platform, 0),
			})
		}
		releases[position].Platforms = append(releases[position].Platforms, platforms[index])
	}
	return releases
}

func normalizePlatforms(report *Report) {
	for releaseIndex := range report.Releases {
		for platformIndex := range report.Releases[releaseIndex].Platforms {
			current := &report.Releases[releaseIndex].Platforms[platformIndex]
			current.Arch = platform.CanonicalArch(current.Arch)
		}
	}
}
