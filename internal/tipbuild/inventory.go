package tipbuild

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qmuntal/gosizehistory/internal/archiveinventory"
	"github.com/qmuntal/gosizehistory/internal/platform"
)

var distributionPackages = []string{
	"cmd/go",
	"cmd/gofmt",
	"cmd/asm",
	"cmd/cgo",
	"cmd/compile",
	"cmd/cover",
	"cmd/fix",
	"cmd/link",
	"cmd/preprofile",
	"cmd/vet",
}

type Target struct {
	OS     string `json:"GOOS"`
	Arch   string `json:"GOARCH"`
	Broken bool   `json:"Broken"`
}

func DistributionTargets(targets []Target, goos, goarch string) []Target {
	selected := make([]Target, 0, len(targets))
	for _, target := range targets {
		if target.Broken || !isStandaloneDistributionTarget(target) {
			continue
		}
		if goos != "" && target.OS != goos {
			continue
		}
		if goarch != "" && platform.CanonicalArch(target.Arch) != platform.CanonicalArch(goarch) {
			continue
		}
		selected = append(selected, target)
	}
	sort.Slice(selected, func(left, right int) bool {
		if selected[left].OS != selected[right].OS {
			return selected[left].OS < selected[right].OS
		}
		return selected[left].Arch < selected[right].Arch
	})
	return selected
}

func isStandaloneDistributionTarget(target Target) bool {
	switch target.OS {
	case "android", "ios", "js", "wasip1":
		return false
	default:
		return true
	}
}

func Inventory(directory string, target Target) ([]archiveinventory.Tool, error) {
	tools := make([]archiveinventory.Tool, 0, len(distributionPackages))
	for _, packagePath := range distributionPackages {
		name := strings.TrimPrefix(packagePath, "cmd/")
		filename := name
		if target.OS == "windows" {
			filename += ".exe"
		}
		fullPath := filepath.Join(directory, filename)
		info, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("inspect %s/%s: %w", target.OS, target.Arch, err)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return nil, fmt.Errorf("inspect %s/%s: %s is not a non-empty regular file", target.OS, target.Arch, filename)
		}

		category := "tool"
		path := filepath.ToSlash(filepath.Join("go", "pkg", "tool", target.OS+"_"+target.Arch, filename))
		if name == "go" || name == "gofmt" {
			category = "command"
			path = filepath.ToSlash(filepath.Join("go", "bin", filename))
		}
		tools = append(tools, archiveinventory.Tool{
			Name:     name,
			Path:     path,
			Category: category,
			Size:     info.Size(),
		})
	}
	return tools, nil
}
