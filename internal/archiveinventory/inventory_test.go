package archiveinventory

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var archiveEntries = []struct {
	name string
	size int
}{
	{name: "go/pkg/tool/linux_amd64/compile", size: 11},
	{name: "go/bin/go", size: 7},
	{name: "go/pkg/tool/linux_amd64/link", size: 13},
	{name: "go/bin/gofmt", size: 5},
	{name: "go/src/cmd/compile/main.go", size: 17},
	{name: "go/pkg/tool/linux_amd64/nested/ignored", size: 19},
}

var expectedTools = []Tool{
	{Name: "go", Path: "go/bin/go", Category: "command", Size: 7},
	{Name: "gofmt", Path: "go/bin/gofmt", Category: "command", Size: 5},
	{Name: "compile", Path: "go/pkg/tool/linux_amd64/compile", Category: "tool", Size: 11},
	{Name: "link", Path: "go/pkg/tool/linux_amd64/link", Category: "tool", Size: 13},
}

func TestInspectTarGz(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "go.test.linux-amd64.tar.gz")
	writeTarGz(t, filename)

	tools, err := Inspect(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tools, expectedTools) {
		t.Fatalf("tools mismatch\ngot:  %#v\nwant: %#v", tools, expectedTools)
	}
}

func TestInspectZip(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "go.test.windows-amd64.zip")
	writeZip(t, filename)

	tools, err := Inspect(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tools, expectedTools) {
		t.Fatalf("tools mismatch\ngot:  %#v\nwant: %#v", tools, expectedTools)
	}
}

func writeTarGz(t *testing.T, filename string) {
	t.Helper()

	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range archiveEntries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o755, Size: int64(entry.size)}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(make([]byte, entry.size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeZip(t *testing.T, filename string) {
	t.Helper()

	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	zipWriter := zip.NewWriter(file)
	for _, entry := range archiveEntries {
		writer, err := zipWriter.Create(entry.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(make([]byte, entry.size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
