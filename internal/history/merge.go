package history

import (
	"encoding/json"
	"fmt"
	goversion "go/version"
	"os"
	"sort"
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
	normalizePlatforms(&report)
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

func MergeStable(base, updates Report) (Report, error) {
	if base.SchemaVersion != SchemaVersion || updates.SchemaVersion != SchemaVersion {
		return Report{}, fmt.Errorf("cannot merge incompatible report schemas")
	}

	baseVersions := make(map[string]string)
	for _, release := range base.Releases {
		if !release.Stable || release.Development {
			continue
		}
		minor, err := releaseMinor(release.Version)
		if err != nil {
			return Report{}, err
		}
		if existing, ok := baseVersions[minor]; ok && existing != release.Version {
			return Report{}, fmt.Errorf("base report contains both %s and %s for %s", existing, release.Version, minor)
		}
		baseVersions[minor] = release.Version
	}

	replacements := make(map[string]Release)
	for _, release := range updates.Releases {
		if !release.Stable || release.Development {
			return Report{}, fmt.Errorf("stable update contains non-stable release %q", release.Version)
		}
		minor, err := releaseMinor(release.Version)
		if err != nil {
			return Report{}, err
		}
		if existing, ok := replacements[minor]; ok {
			return Report{}, fmt.Errorf("stable update contains both %s and %s for %s", existing.Version, release.Version, minor)
		}
		if current := baseVersions[minor]; current != "" && goversion.Compare(release.Version, current) <= 0 {
			return Report{}, fmt.Errorf("stable update %s does not advance %s", release.Version, current)
		}
		replacements[minor] = release
	}
	if len(replacements) == 0 {
		return Report{}, fmt.Errorf("stable update contains no releases")
	}

	releases := make([]Release, 0, len(base.Releases)+len(replacements))
	development := make([]Release, 0, 1)
	for _, release := range base.Releases {
		if release.Development {
			development = append(development, release)
			continue
		}
		if release.Stable {
			minor, err := releaseMinor(release.Version)
			if err != nil {
				return Report{}, err
			}
			if _, replaced := replacements[minor]; replaced {
				continue
			}
		}
		releases = append(releases, release)
	}
	for _, release := range replacements {
		releases = append(releases, release)
	}
	sort.SliceStable(releases, func(left, right int) bool {
		return goversion.Compare(releases[left].Version, releases[right].Version) > 0
	})
	releases = append(releases, development...)

	generatedAt := base.GeneratedAt
	if updates.GeneratedAt.After(generatedAt) {
		generatedAt = updates.GeneratedAt
	}
	return Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   generatedAt,
		Source:        base.Source,
		Releases:      releases,
	}, nil
}

func releaseMinor(version string) (string, error) {
	if !goversion.IsValid(version) {
		return "", fmt.Errorf("invalid Go release version %q", version)
	}
	minor := goversion.Lang(version)
	if !strings.Contains(strings.TrimPrefix(minor, "go"), ".") {
		return "", fmt.Errorf("Go release version %q has no minor line", version)
	}
	return minor, nil
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
