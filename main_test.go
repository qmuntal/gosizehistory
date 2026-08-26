package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qmuntal/gosizehistory/internal/goreleases"
	"github.com/qmuntal/gosizehistory/internal/history"
)

func TestRunDryRunWithLocalMetadata(t *testing.T) {
	metadata := `[{"version":"go1.2.3","stable":true,"files":[
		{"filename":"go1.2.3.src.tar.gz","kind":"source"},
		{"filename":"go1.2.3.linux-amd64.tar.gz","os":"linux","arch":"amd64","size":2048,"kind":"archive"}
	]}]`
	filename := filepath.Join(t.TempDir(), "dl.json")
	if err := os.WriteFile(filename, []byte(metadata), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(context.Background(), []string{
		"-metadata", filename,
		"-version", "go1.2.3",
		"-os", "linux",
		"-dry-run",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "selected 1 archives across 1 releases (2.00 KiB)") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunRejectsInvalidWorkerCount(t *testing.T) {
	err := run(context.Background(), []string{"-workers", "0"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || err.Error() != "workers must be at least 1" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDryRunLatestPerMinor(t *testing.T) {
	metadata := `[
		{"version":"go1.3rc1","stable":false,"files":[{"filename":"go1.3rc1.zip","size":8192,"kind":"archive"}]},
		{"version":"go1.2.4","stable":true,"files":[{"filename":"go1.2.4.zip","size":4096,"kind":"archive"}]},
		{"version":"go1.2.3","stable":true,"files":[{"filename":"go1.2.3.zip","size":2048,"kind":"archive"}]},
		{"version":"go1.1.2","stable":true,"files":[{"filename":"go1.1.2.zip","size":1024,"kind":"archive"}]}
	]`
	filename := filepath.Join(t.TempDir(), "dl.json")
	if err := os.WriteFile(filename, []byte(metadata), 0o600); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	err := run(context.Background(), []string{"-metadata", filename, "-latest-per-minor", "-dry-run"}, &bytes.Buffer{}, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "selected 2 archives across 2 releases (5.00 KiB)") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunRejectsLatestPerMinorWithVersion(t *testing.T) {
	err := run(context.Background(), []string{"-latest-per-minor", "-version", "go1.2.3"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || err.Error() != "latest-per-minor cannot be combined with version" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRefreshStableDryRunSelectsOnlyNewVersions(t *testing.T) {
	directory := t.TempDir()
	metadataPath := filepath.Join(directory, "dl.json")
	reportPath := filepath.Join(directory, "report.json")
	metadata := `[
		{"version":"go1.3.1","stable":true,"files":[{"filename":"go1.3.1.linux-amd64.tar.gz","os":"linux","arch":"amd64","size":4096,"kind":"archive"}]},
		{"version":"go1.2.4","stable":true,"files":[{"filename":"go1.2.4.linux-amd64.tar.gz","os":"linux","arch":"amd64","size":2048,"kind":"archive"}]}
	]`
	if err := os.WriteFile(metadataPath, []byte(metadata), 0o600); err != nil {
		t.Fatal(err)
	}
	base := history.Report{
		SchemaVersion: history.SchemaVersion,
		Releases:      []history.Release{{Version: "go1.2.4", Stable: true}},
	}
	if err := history.WriteFile(reportPath, base); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	err := run(context.Background(), []string{
		"-metadata", metadataPath,
		"-refresh-stable",
		"-dry-run",
		"-output", reportPath,
	}, &bytes.Buffer{}, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "selected 1 archives across 1 releases (4.00 KiB)") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunRefreshStableNoop(t *testing.T) {
	directory := t.TempDir()
	metadataPath := filepath.Join(directory, "dl.json")
	reportPath := filepath.Join(directory, "report.json")
	metadata := `[{"version":"go1.2.4","stable":true,"files":[{"filename":"go1.2.4.linux-amd64.tar.gz","os":"linux","arch":"amd64","size":2048,"kind":"archive"}]}]`
	if err := os.WriteFile(metadataPath, []byte(metadata), 0o600); err != nil {
		t.Fatal(err)
	}
	base := history.Report{SchemaVersion: history.SchemaVersion, Releases: []history.Release{{Version: "go1.2.4", Stable: true}}}
	if err := history.WriteFile(reportPath, base); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	if err := run(context.Background(), []string{
		"-metadata", metadataPath,
		"-refresh-stable",
		"-output", reportPath,
	}, &bytes.Buffer{}, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "no new stable releases") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestNewStablePlanSelectsOnlyNewerMinorVersions(t *testing.T) {
	plan := history.Plan{Archives: []goreleases.Archive{
		{Release: "go1.28.0", File: goreleases.File{Filename: "go1.28.0.linux-amd64.tar.gz"}},
		{Release: "go1.27.1", File: goreleases.File{Filename: "go1.27.1.linux-amd64.tar.gz"}},
		{Release: "go1.27.1", File: goreleases.File{Filename: "go1.27.1.windows-amd64.zip"}},
		{Release: "go1.26.6", File: goreleases.File{Filename: "go1.26.6.linux-amd64.tar.gz"}},
	}}
	existing := history.Report{Releases: []history.Release{
		{Version: "go1.27.0", Stable: true},
		{Version: "go1.26.7", Stable: true},
	}}

	got := newStablePlan(plan, existing)
	want := []string{
		"go1.28.0.linux-amd64.tar.gz",
		"go1.27.1.linux-amd64.tar.gz",
		"go1.27.1.windows-amd64.zip",
	}
	if len(got.Archives) != len(want) {
		t.Fatalf("got %d archives, want %d: %#v", len(got.Archives), len(want), got.Archives)
	}
	for index, filename := range want {
		if got.Archives[index].File.Filename != filename {
			t.Errorf("archive %d = %q, want %q", index, got.Archives[index].File.Filename, filename)
		}
	}
}

func TestRunRefreshStableMergesNewPatch(t *testing.T) {
	directory := t.TempDir()
	reportPath := filepath.Join(directory, "report.json")
	archive := testReleaseZip(t)
	metadata := fmt.Sprintf(`[
		{"version":"go1.2.4","stable":true,"files":[{"filename":"go1.2.4.windows-amd64.zip","os":"windows","arch":"amd64","size":%d,"kind":"archive"}]},
		{"version":"go1.1.2","stable":true,"files":[{"filename":"go1.1.2.windows-amd64.zip","os":"windows","arch":"amd64","size":1,"kind":"archive"}]}
	]`, len(archive))
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/metadata":
			_, _ = response.Write([]byte(metadata))
		case "/dl/go1.2.4.windows-amd64.zip":
			_, _ = response.Write(archive)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	base := history.Report{
		SchemaVersion: history.SchemaVersion,
		Source:        "original-metadata.json",
		Releases: []history.Release{
			{Version: "go1.2.3", Stable: true},
			{Version: "go1.1.2", Stable: true, Platforms: []history.Platform{{OS: "preserved", Arch: "amd64"}}},
			{Version: "go1.3-tip", Development: true, Revision: "tip-revision"},
		},
	}
	if err := history.WriteFile(reportPath, base); err != nil {
		t.Fatal(err)
	}

	if err := run(context.Background(), []string{
		"-metadata", server.URL + "/metadata",
		"-download-base-url", server.URL + "/dl/",
		"-cache-dir", filepath.Join(directory, "cache"),
		"-refresh-stable",
		"-workers", "1",
		"-retries", "1",
		"-output", reportPath,
	}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	got, err := history.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	wantVersions := []string{"go1.2.4", "go1.1.2", "go1.3-tip"}
	for index, want := range wantVersions {
		if got.Releases[index].Version != want {
			t.Fatalf("release %d = %q, want %q", index, got.Releases[index].Version, want)
		}
	}
	if len(got.Releases[0].Platforms) != 1 || len(got.Releases[0].Platforms[0].Tools) != 2 {
		t.Fatalf("new stable release was not measured: %#v", got.Releases[0])
	}
	if got.Releases[1].Platforms[0].OS != "preserved" {
		t.Fatalf("untouched stable release was not preserved: %#v", got.Releases[1])
	}
	if got.Releases[2].Revision != "tip-revision" || !got.Releases[2].Development {
		t.Fatalf("tip release was not preserved: %#v", got.Releases[2])
	}
	if got.Source != base.Source {
		t.Fatalf("source = %q, want %q", got.Source, base.Source)
	}
}

func testReleaseZip(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for path, size := range map[string]int{
		"go/bin/go.exe":                         13,
		"go/pkg/tool/windows_amd64/compile.exe": 17,
	} {
		file, err := writer.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(make([]byte, size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestRunRejectsInvalidTipOptions(t *testing.T) {
	for _, args := range [][]string{
		{"-tip", "-latest-per-minor"},
		{"-tip", "-version", "go1.2.3"},
		{"-tip", "-dry-run"},
		{"-tip", "-merge-tip-report", "tip.json"},
		{"-merge-tip-report", "tip.json", "-dry-run"},
		{"-tip-workers", "0"},
	} {
		err := run(context.Background(), args, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil {
			t.Fatalf("run(%v) unexpectedly succeeded", args)
		}
	}
}

func TestRunRejectsInvalidRefreshStableOptions(t *testing.T) {
	for _, args := range [][]string{
		{"-refresh-stable", "-latest-per-minor"},
		{"-refresh-stable", "-version", "go1.27.0"},
		{"-refresh-stable", "-os", "linux"},
		{"-refresh-stable", "-arch", "amd64"},
		{"-refresh-stable", "-tip"},
		{"-refresh-stable", "-merge-tip-report", "tip.json"},
		{"-refresh-stable", "-output", "-"},
	} {
		if err := run(context.Background(), args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatalf("run(%v) unexpectedly succeeded", args)
		}
	}
}

func TestRunMergeTipReport(t *testing.T) {
	directory := t.TempDir()
	basePath := filepath.Join(directory, "report.json")
	tipPath := filepath.Join(directory, "tip.json")
	base := history.Report{
		SchemaVersion: history.SchemaVersion,
		Source:        "dl.json",
		Releases:      []history.Release{{Version: "go1.27.0", Stable: true}},
	}
	tip := history.NewDevelopmentReport(time.Now(), "tip", history.Release{Version: "go1.28-tip", Revision: "abc"})
	if err := history.WriteFile(basePath, base); err != nil {
		t.Fatal(err)
	}
	if err := history.WriteFile(tipPath, tip); err != nil {
		t.Fatal(err)
	}

	if err := run(context.Background(), []string{
		"-merge-tip-report", tipPath,
		"-output", basePath,
	}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	merged, err := history.ReadFile(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Releases) != 2 || !merged.Releases[1].Development {
		t.Fatalf("unexpected merged report: %#v", merged)
	}
}

func TestMergeAndWriteReport(t *testing.T) {
	basePath := filepath.Join(t.TempDir(), "report.json")
	base := history.Report{
		SchemaVersion: history.SchemaVersion,
		Source:        "dl.json",
		Releases:      []history.Release{{Version: "go1.27.0", Stable: true}},
	}
	if err := history.WriteFile(basePath, base); err != nil {
		t.Fatal(err)
	}
	tip := history.NewDevelopmentReport(time.Now(), "tip", history.Release{Version: "go1.28-tip", Revision: "abc"})

	if err := mergeAndWriteReport(basePath, "", tip, &bytes.Buffer{}, log.New(&bytes.Buffer{}, "", 0)); err != nil {
		t.Fatal(err)
	}
	merged, err := history.ReadFile(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Releases) != 2 || merged.Releases[1].Version != "go1.28-tip" {
		t.Fatalf("unexpected merged releases: %#v", merged.Releases)
	}
}

func TestFormatBytes(t *testing.T) {
	if got := formatBytes(5 * 1024 * 1024); got != "5.00 MiB" {
		t.Fatalf("formatBytes returned %q", got)
	}
}
