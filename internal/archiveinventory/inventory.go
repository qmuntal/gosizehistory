package archiveinventory

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"sort"
	"strings"
)

type Tool struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Category string `json:"category"`
	Size     int64  `json:"size"`
}

func Inspect(filename string) ([]Tool, error) {
	switch {
	case strings.HasSuffix(filename, ".tar.gz"):
		return inspectTarGz(filename)
	case strings.HasSuffix(filename, ".zip"):
		return inspectZip(filename)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", filename)
	}
}

func inspectTarGz(filename string) ([]Tool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("open gzip stream: %w", err)
	}
	defer gzipReader.Close()

	var tools []Tool
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar entry: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}
		if tool, ok := classify(header.Name, header.Size); ok {
			tools = append(tools, tool)
		}
	}

	sortTools(tools)
	return tools, nil
}

func inspectZip(filename string) ([]Tool, error) {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var tools []Tool
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > math.MaxInt64 {
			return nil, fmt.Errorf("zip entry %q is too large", file.Name)
		}
		if tool, ok := classify(file.Name, int64(file.UncompressedSize64)); ok {
			tools = append(tools, tool)
		}
	}

	sortTools(tools)
	return tools, nil
}

func classify(filename string, size int64) (Tool, bool) {
	filename = strings.TrimPrefix(path.Clean(strings.ReplaceAll(filename, "\\", "/")), "./")
	parts := strings.Split(filename, "/")

	category := ""
	switch {
	case len(parts) == 3 && parts[0] == "go" && parts[1] == "bin":
		category = "command"
	case len(parts) == 5 && parts[0] == "go" && parts[1] == "pkg" && parts[2] == "tool":
		category = "tool"
	default:
		return Tool{}, false
	}

	name := strings.TrimSuffix(parts[len(parts)-1], ".exe")
	return Tool{Name: name, Path: filename, Category: category, Size: size}, true
}

func sortTools(tools []Tool) {
	sort.Slice(tools, func(left, right int) bool {
		if tools[left].Category != tools[right].Category {
			return tools[left].Category < tools[right].Category
		}
		if tools[left].Name != tools[right].Name {
			return tools[left].Name < tools[right].Name
		}
		return tools[left].Path < tools[right].Path
	})
}
