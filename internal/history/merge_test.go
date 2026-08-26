package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMergeDevelopmentReplacesPreviousTip(t *testing.T) {
	stableTime := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	tipTime := stableTime.Add(time.Hour)
	base := Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   stableTime,
		Source:        "dl.json",
		Releases: []Release{
			{Version: "go1.27.0", Stable: true},
			{Version: "go1.28-tip", Development: true, Revision: "old"},
		},
	}
	tip := NewDevelopmentReport(tipTime, "https://github.com/golang/go.git@new", Release{
		Version:  "go1.28-tip",
		Revision: "new",
	})

	merged, err := MergeDevelopment(base, tip)
	if err != nil {
		t.Fatal(err)
	}
	if merged.Source != "dl.json" || !merged.GeneratedAt.Equal(tipTime) {
		t.Fatalf("unexpected merged metadata: %#v", merged)
	}
	if len(merged.Releases) != 2 || merged.Releases[0].Version != "go1.27.0" || merged.Releases[1].Revision != "new" {
		t.Fatalf("unexpected merged releases: %#v", merged.Releases)
	}
	if merged.Releases[1].Source != "https://github.com/golang/go.git" {
		t.Fatalf("tip source = %q", merged.Releases[1].Source)
	}
}

func TestMergeStableReplacesOnlyUpdatedMinorLines(t *testing.T) {
	stableTime := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	updateTime := stableTime.Add(time.Hour)
	base := Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   stableTime,
		Source:        "dl.json",
		Releases: []Release{
			{Version: "go1.27.0", Stable: true, Platforms: []Platform{{OS: "linux", Arch: "amd64"}}},
			{Version: "go1.26.7", Stable: true, Platforms: []Platform{{OS: "windows", Arch: "amd64"}}},
			{Version: "go1.28-tip", Development: true, Revision: "tip-revision"},
		},
	}
	updates := Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   updateTime,
		Source:        "new-metadata.json",
		Releases: []Release{
			{Version: "go1.28.0", Stable: true},
			{Version: "go1.27.1", Stable: true},
		},
	}

	merged, err := MergeStable(base, updates)
	if err != nil {
		t.Fatal(err)
	}
	wantVersions := []string{"go1.28.0", "go1.27.1", "go1.26.7", "go1.28-tip"}
	if len(merged.Releases) != len(wantVersions) {
		t.Fatalf("got %d releases, want %d: %#v", len(merged.Releases), len(wantVersions), merged.Releases)
	}
	for index, want := range wantVersions {
		if merged.Releases[index].Version != want {
			t.Fatalf("release %d = %q, want %q", index, merged.Releases[index].Version, want)
		}
	}
	if merged.Releases[2].Platforms[0].OS != "windows" {
		t.Fatalf("untouched release data was not preserved: %#v", merged.Releases[2])
	}
	if merged.Releases[3].Revision != "tip-revision" || !merged.Releases[3].Development {
		t.Fatalf("development release was not preserved: %#v", merged.Releases[3])
	}
	if merged.Source != base.Source || !merged.GeneratedAt.Equal(updateTime) {
		t.Fatalf("unexpected merged metadata: %#v", merged)
	}
}

func TestMergeStableRejectsDuplicateMinorUpdates(t *testing.T) {
	base := Report{SchemaVersion: SchemaVersion, Releases: []Release{{Version: "go1.26.7", Stable: true}}}
	updates := Report{
		SchemaVersion: SchemaVersion,
		Releases: []Release{
			{Version: "go1.27.1", Stable: true},
			{Version: "go1.27.2", Stable: true},
		},
	}
	if _, err := MergeStable(base, updates); err == nil {
		t.Fatal("MergeStable accepted duplicate minor updates")
	}
}

func TestMergeStableRejectsDowngrade(t *testing.T) {
	base := Report{SchemaVersion: SchemaVersion, Releases: []Release{{Version: "go1.27.2", Stable: true}}}
	updates := Report{SchemaVersion: SchemaVersion, Releases: []Release{{Version: "go1.27.1", Stable: true}}}
	if _, err := MergeStable(base, updates); err == nil {
		t.Fatal("MergeStable accepted a stable release downgrade")
	}
}

func TestReadFile(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "report.json")
	report := Report{SchemaVersion: SchemaVersion, Releases: []Release{{
		Version: "go1.27.0",
		Platforms: []Platform{{
			OS:      "linux",
			Arch:    "armv6l",
			Archive: &Archive{Filename: "go1.27.0.linux-armv6l.tar.gz"},
		}},
	}}}
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(file, report); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != SchemaVersion || len(got.Releases) != 1 || got.Releases[0].Version != "go1.27.0" {
		t.Fatalf("unexpected report: %#v", got)
	}
	platform := got.Releases[0].Platforms[0]
	if platform.Arch != "arm" || platform.Archive.Filename != "go1.27.0.linux-armv6l.tar.gz" {
		t.Fatalf("unexpected normalized platform: %#v", platform)
	}
}

func TestPreserveDevelopment(t *testing.T) {
	base := Report{SchemaVersion: SchemaVersion, Releases: []Release{{Version: "go1.27.0", Stable: true}}}
	tip := Report{
		SchemaVersion: SchemaVersion,
		Releases:      []Release{{Version: "go1.28-tip", Development: true, Revision: "abc"}},
	}
	merged, err := PreserveDevelopment(base, tip)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Releases) != 2 || merged.Releases[1].Revision != "abc" {
		t.Fatalf("unexpected merged releases: %#v", merged.Releases)
	}

	unchanged, err := PreserveDevelopment(base, Report{SchemaVersion: SchemaVersion, Releases: []Release{{Version: "go1.26.7"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(unchanged.Releases) != 1 || unchanged.Releases[0].Version != "go1.27.0" {
		t.Fatalf("unexpected unchanged report: %#v", unchanged)
	}
}
