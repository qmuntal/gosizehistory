package history

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func ReadFile(filename string) (Report, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Report{}, fmt.Errorf("open report: %w", err)
	}
	defer file.Close()

	var report Report
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&report); err != nil {
		return Report{}, fmt.Errorf("decode report: %w", err)
	}
	if report.SchemaVersion != SchemaVersion {
		return Report{}, fmt.Errorf("unsupported report schema version %d", report.SchemaVersion)
	}
	if len(report.Releases) == 0 {
		return Report{}, fmt.Errorf("report has no releases")
	}
	return report, nil
}

func MergeDevelopment(base, development Report) (Report, error) {
	if base.SchemaVersion != SchemaVersion || development.SchemaVersion != SchemaVersion {
		return Report{}, fmt.Errorf("cannot merge incompatible report schemas")
	}

	developmentReleases := make([]Release, 0, 1)
	for _, release := range development.Releases {
		if release.Development {
			developmentReleases = append(developmentReleases, release)
		}
	}
	if len(developmentReleases) != 1 {
		return Report{}, fmt.Errorf("development report must contain exactly one development release, got %d", len(developmentReleases))
	}

	releases := make([]Release, 0, len(base.Releases)+1)
	for _, release := range base.Releases {
		if !release.Development {
			releases = append(releases, release)
		}
	}
	developmentRelease := developmentReleases[0]
	if developmentRelease.Source == "" {
		developmentRelease.Source = development.Source
		if separator := strings.LastIndex(developmentRelease.Source, "@"); separator >= 0 {
			developmentRelease.Source = developmentRelease.Source[:separator]
		}
	}
	releases = append(releases, developmentRelease)

	generatedAt := base.GeneratedAt
	if development.GeneratedAt.After(generatedAt) {
		generatedAt = development.GeneratedAt
	}
	return Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   generatedAt,
		Source:        base.Source,
		Releases:      releases,
	}, nil
}

func PreserveDevelopment(base, existing Report) (Report, error) {
	developmentCount := 0
	for _, release := range existing.Releases {
		if release.Development {
			developmentCount++
		}
	}
	switch developmentCount {
	case 0:
		return base, nil
	case 1:
		return MergeDevelopment(base, existing)
	default:
		return Report{}, fmt.Errorf("existing report contains %d development releases", developmentCount)
	}
}

func NewDevelopmentReport(generatedAt time.Time, source string, release Release) Report {
	release.Development = true
	return Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   generatedAt,
		Source:        source,
		Releases:      []Release{release},
	}
}
