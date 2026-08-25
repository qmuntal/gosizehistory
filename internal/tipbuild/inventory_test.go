package tipbuild

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDistributionTargets(t *testing.T) {
	targets := []Target{
		{OS: "windows", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "android", Arch: "arm64"},
		{OS: "ios", Arch: "arm64"},
		{OS: "js", Arch: "wasm"},
		{OS: "wasip1", Arch: "wasm"},
		{OS: "broken", Arch: "port", Broken: true},
		{OS: "linux", Arch: "amd64"},
	}

	selected := DistributionTargets(targets, "", "")
	want := []Target{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
	}
	if !reflect.DeepEqual(selected, want) {
		t.Fatalf("targets mismatch\ngot:  %#v\nwant: %#v", selected, want)
	}

	filtered := DistributionTargets(targets, "linux", "arm64")
	if len(filtered) != 1 || filtered[0].OS != "linux" || filtered[0].Arch != "arm64" {
		t.Fatalf("unexpected filtered targets: %#v", filtered)
	}
}

func TestInventory(t *testing.T) {
	directory := t.TempDir()
	for index, packagePath := range distributionPackages {
		name := filepath.Base(packagePath) + ".exe"
		if err := os.WriteFile(filepath.Join(directory, name), make([]byte, index+1), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	tools, err := Inventory(directory, Target{OS: "windows", Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != len(distributionPackages) {
		t.Fatalf("got %d tools, want %d", len(tools), len(distributionPackages))
	}
	if tools[0].Name != "go" || tools[0].Category != "command" || tools[0].Path != "go/bin/go.exe" || tools[0].Size != 1 {
		t.Fatalf("unexpected go command: %#v", tools[0])
	}
	if tools[2].Name != "asm" || tools[2].Category != "tool" || tools[2].Path != "go/pkg/tool/windows_amd64/asm.exe" || tools[2].Size != 3 {
		t.Fatalf("unexpected asm tool: %#v", tools[2])
	}
}
