package history

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/qmuntal/gosizehistory/internal/archiveinventory"
	"github.com/qmuntal/gosizehistory/internal/goreleases"
)

func TestBuildFromHTTPAndWriteFile(t *testing.T) {
	tarGz := testTarGz(t, "go/bin/go", 7, "go/pkg/tool/linux_amd64/compile", 11)
	zipArchive := testZip(t, "go/bin/go.exe", 13, "go/pkg/tool/windows_amd64/compile.exe", 17)
	tarDigest := sha256.Sum256(tarGz)
	zipDigest := sha256.Sum256(zipArchive)

	metadata := fmt.Sprintf(`[{"version":"go1.2.3","stable":true,"files":[
		{"filename":"go1.2.3.linux-amd64.tar.gz","os":"linux","arch":"amd64","size":%d,"sha256":"%s","kind":"archive"},
		{"filename":"go1.2.3.windows-amd64.zip","os":"windows","arch":"amd64","size":%d,"sha256":"%s","kind":"archive"}
	]}]`, len(tarGz), hex.EncodeToString(tarDigest[:]), len(zipArchive), hex.EncodeToString(zipDigest[:]))

	archives := map[string][]byte{
		"/dl/go1.2.3.linux-amd64.tar.gz": tarGz,
		"/dl/go1.2.3.windows-amd64.zip":  zipArchive,
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/metadata" {
			_, _ = response.Write([]byte(metadata))
			return
		}
		contents, ok := archives[request.URL.Path]
		if !ok {
			http.NotFound(response, request)
			return
		}
		_, _ = response.Write(contents)
	}))
	defer server.Close()

	plan, err := NewPlan(context.Background(), server.Client(), server.URL+"/metadata", goreleases.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ReleaseCount() != 1 {
		t.Fatalf("got %d releases, want 1", plan.ReleaseCount())
	}
	knownSize, unknown := plan.KnownDownloadSize()
	if knownSize != int64(len(tarGz)+len(zipArchive)) || unknown != 0 {
		t.Fatalf("unexpected plan size: %d bytes, %d unknown", knownSize, unknown)
	}

	fixedTime := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.FixedZone("test", 3600))
	var completed atomic.Int32
	report, err := Build(context.Background(), plan, BuildConfig{
		CacheDir: t.TempDir(),
		BaseURL:  server.URL + "/dl/",
		Client:   server.Client(),
		Workers:  2,
		Retries:  1,
		Now:      func() time.Time { return fixedTime },
		Progress: func(progress Progress) { completed.Add(1) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Load() != 2 {
		t.Fatalf("got %d progress events, want 2", completed.Load())
	}
	if report.GeneratedAt.Location() != time.UTC || !report.GeneratedAt.Equal(fixedTime) {
		t.Fatalf("unexpected report timestamp %v", report.GeneratedAt)
	}
	if len(report.Releases) != 1 || len(report.Releases[0].Platforms) != 2 {
		t.Fatalf("unexpected report shape: %#v", report.Releases)
	}
	linux := report.Releases[0].Platforms[0]
	if linux.Tools[0].Name != "go" || linux.Tools[0].Size != 7 || linux.Tools[1].Name != "compile" {
		t.Fatalf("unexpected Linux tools: %#v", linux.Tools)
	}
	windows := report.Releases[0].Platforms[1]
	if windows.Tools[0].Name != "go" || windows.Tools[1].Name != "compile" {
		t.Fatalf("Windows tool names were not normalized: %#v", windows.Tools)
	}

	output := filepath.Join(t.TempDir(), "reports", "sizes.json")
	if err := WriteFile(output, report); err != nil {
		t.Fatal(err)
	}
	var decoded Report
	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SchemaVersion != SchemaVersion || len(decoded.Releases) != 1 {
		t.Fatalf("unexpected decoded report: %#v", decoded)
	}
}

func TestWriteDevelopmentReleaseOmitsArchive(t *testing.T) {
	commitTime := time.Date(2026, time.August, 25, 7, 7, 23, 0, time.UTC)
	report := Report{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   commitTime,
		Source:        "https://github.com/golang/go.git@abcdef",
		Releases: []Release{{
			Version:     "go1.28-tip",
			Development: true,
			Revision:    "abcdef",
			CommitTime:  &commitTime,
			Platforms: []Platform{{
				OS:    "linux",
				Arch:  "amd64",
				Tools: []archiveinventory.Tool{},
			}},
		}},
	}

	var buffer bytes.Buffer
	if err := Write(&buffer, report); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buffer.Bytes(), []byte(`"archive"`)) {
		t.Fatalf("development report contains archive metadata: %s", buffer.String())
	}
	if !bytes.Contains(buffer.Bytes(), []byte(`"development": true`)) || !bytes.Contains(buffer.Bytes(), []byte(`"revision": "abcdef"`)) {
		t.Fatalf("development metadata is missing: %s", buffer.String())
	}
}
func testTarGz(t *testing.T, entries ...any) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for index := 0; index < len(entries); index += 2 {
		name := entries[index].(string)
		size := entries[index+1].(int)
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(size)}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(make([]byte, size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func testZip(t *testing.T, entries ...any) []byte {
	t.Helper()
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	for index := 0; index < len(entries); index += 2 {
		name := entries[index].(string)
		size := entries[index+1].(int)
		writer, err := zipWriter.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(make([]byte, size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
