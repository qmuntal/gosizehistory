package goreleases

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const testMetadata = `[
  {
    "version": "go1.2.3",
    "stable": true,
    "files": [
      {"filename":"go1.2.3.src.tar.gz","kind":"source"},
      {"filename":"go1.2.3.linux-amd64.tar.gz","os":"linux","arch":"amd64","size":10,"kind":"archive"},
      {"filename":"go1.2.3.windows-amd64.zip","os":"windows","arch":"amd64","size":11,"kind":"archive"},
      {"filename":"go1.2.3.windows-amd64.msi","os":"windows","arch":"amd64","size":12,"kind":"installer"}
    ]
  },
  {
    "version": "go1.2rc1",
    "stable": false,
    "files": [
      {"filename":"go1.2rc1.linux-amd64.tar.gz","os":"linux","arch":"amd64","size":9,"kind":"archive"}
    ]
  }
]`

func TestLoadFromFileAndSelectArchives(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "dl.json")
	if err := os.WriteFile(filename, []byte(testMetadata), 0o600); err != nil {
		t.Fatal(err)
	}

	releases, err := Load(context.Background(), http.DefaultClient, filename)
	if err != nil {
		t.Fatal(err)
	}
	archives := SelectArchives(releases, Filter{Version: "go1.2.3", Arch: "amd64"})
	want := []string{"go1.2.3.linux-amd64.tar.gz", "go1.2.3.windows-amd64.zip"}
	got := make([]string, len(archives))
	for index := range archives {
		got[index] = archives[index].File.Filename
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("archive filenames mismatch: got %v, want %v", got, want)
	}
}

func TestLoadFromHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.UserAgent() != "gosizehistory/1" {
			t.Errorf("unexpected user agent %q", request.UserAgent())
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(testMetadata))
	}))
	defer server.Close()

	releases, err := Load(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 2 {
		t.Fatalf("got %d releases, want 2", len(releases))
	}
}

func TestSelectArchivesLatestPerMinor(t *testing.T) {
	releases := []Release{
		{Version: "go1.3rc1", Stable: false, Files: []File{{Filename: "go1.3rc1.zip", Kind: "archive"}}},
		{Version: "go1.2.10", Stable: true, Files: []File{{Filename: "go1.2.10.zip", OS: "windows", Kind: "archive"}}},
		{Version: "go1.2.9", Stable: true, Files: []File{{Filename: "go1.2.9.zip", OS: "windows", Kind: "archive"}}},
		{Version: "go1.1.2", Stable: true, Files: []File{{Filename: "go1.1.2.zip", OS: "windows", Kind: "archive"}}},
		{Version: "go1", Stable: true, Files: []File{{Filename: "bootstrap.tar.gz", Kind: "archive"}}},
	}

	archives := SelectArchives(releases, Filter{LatestPerMinor: true, OS: "windows"})
	want := []string{"go1.2.10.zip", "go1.1.2.zip"}
	got := make([]string, len(archives))
	for index := range archives {
		got[index] = archives[index].File.Filename
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("archive filenames mismatch: got %v, want %v", got, want)
	}
}

func TestSelectArchivesTreatsARMAndARMv6LAsAliases(t *testing.T) {
	releases := []Release{{
		Version: "go1.27.0",
		Stable:  true,
		Files: []File{
			{Filename: "go1.27.0.linux-armv6l.tar.gz", OS: "linux", Arch: "armv6l", Kind: "archive"},
			{Filename: "go1.27.0.freebsd-arm.tar.gz", OS: "freebsd", Arch: "arm", Kind: "archive"},
			{Filename: "go1.27.0.linux-arm64.tar.gz", OS: "linux", Arch: "arm64", Kind: "archive"},
		},
	}}
	want := []string{"go1.27.0.linux-armv6l.tar.gz", "go1.27.0.freebsd-arm.tar.gz"}
	for _, arch := range []string{"arm", "armv6", "armv6l"} {
		archives := SelectArchives(releases, Filter{Arch: arch})
		got := make([]string, len(archives))
		for index := range archives {
			got[index] = archives[index].File.Filename
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("arch %q selected %v, want %v", arch, got, want)
		}
	}
}
