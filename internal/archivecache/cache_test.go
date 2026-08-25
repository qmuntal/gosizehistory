package archivecache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestGetDownloadsAndReusesVerifiedArchive(t *testing.T) {
	contents := []byte("archive contents")
	digest := sha256.Sum256(contents)
	artifact := Artifact{
		Filename: "go1.2.3.linux-amd64.tar.gz",
		Size:     int64(len(contents)),
		SHA256:   hex.EncodeToString(digest[:]),
	}

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.URL.Path != "/dl/"+artifact.Filename {
			t.Errorf("unexpected request path %q", request.URL.Path)
		}
		_, _ = response.Write(contents)
	}))
	defer server.Close()

	cache := Cache{Dir: t.TempDir(), BaseURL: server.URL + "/dl/", Client: server.Client(), Retries: 1}
	first, err := cache.Get(context.Background(), artifact)
	if err != nil {
		t.Fatal(err)
	}
	if first.Cached {
		t.Fatal("first result unexpectedly came from cache")
	}
	second, err := cache.Get(context.Background(), artifact)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Cached {
		t.Fatal("second result did not come from cache")
	}
	if requests.Load() != 1 {
		t.Fatalf("got %d HTTP requests, want 1", requests.Load())
	}
	if second.Size != int64(len(contents)) || second.SHA256 != artifact.SHA256 {
		t.Fatalf("unexpected cached result: %#v", second)
	}
}

func TestGetReplacesCorruptCacheEntry(t *testing.T) {
	contents := []byte("valid archive")
	digest := sha256.Sum256(contents)
	artifact := Artifact{
		Filename: "go1.2.3.windows-amd64.zip",
		Size:     int64(len(contents)),
		SHA256:   hex.EncodeToString(digest[:]),
	}

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write(contents)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	target := filepath.Join(cacheDir, artifact.Filename)
	if err := os.WriteFile(target, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}

	cache := Cache{Dir: cacheDir, BaseURL: server.URL, Client: server.Client(), Retries: 1}
	result, err := cache.Get(context.Background(), artifact)
	if err != nil {
		t.Fatal(err)
	}
	if result.Cached {
		t.Fatal("corrupt entry was treated as cached")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(contents) {
		t.Fatalf("cached contents = %q, want %q", got, contents)
	}
}

func TestGetAcceptsLegacyArtifactWithoutIntegrityMetadata(t *testing.T) {
	contents := []byte("legacy archive")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write(contents)
	}))
	defer server.Close()

	cache := Cache{Dir: t.TempDir(), BaseURL: server.URL, Client: server.Client(), Retries: 1}
	result, err := cache.Get(context.Background(), Artifact{Filename: "go1.2.darwin-amd64.tar.gz"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Size != int64(len(contents)) || len(result.SHA256) != sha256.Size*2 {
		t.Fatalf("legacy result was not enriched: %#v", result)
	}
}
