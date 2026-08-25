package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestFormatBytes(t *testing.T) {
	if got := formatBytes(5 * 1024 * 1024); got != "5.00 MiB" {
		t.Fatalf("formatBytes returned %q", got)
	}
}
