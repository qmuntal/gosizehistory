package tipbuild

import (
	"strings"
	"testing"
)

func TestNormalizeDevelopmentVersion(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "devel go1.28-abcdef", want: "go1.28-tip"},
		{input: "go1.28.1", want: "go1.28-tip"},
	} {
		got, err := normalizeDevelopmentVersion(test.input)
		if err != nil {
			t.Fatalf("normalizeDevelopmentVersion(%q): %v", test.input, err)
		}
		if got != test.want {
			t.Fatalf("normalizeDevelopmentVersion(%q) = %q, want %q", test.input, got, test.want)
		}
	}
	if _, err := normalizeDevelopmentVersion("unknown"); err == nil {
		t.Fatal("normalizeDevelopmentVersion accepted an invalid version")
	}
}

func TestCleanEnvironment(t *testing.T) {
	t.Setenv("GOOS", "windows")
	t.Setenv("TEST_TIP_BUILD", "preserved")
	environment := cleanEnvironment(map[string]string{"GOARCH": "arm64"}, "GOOS")
	values := make(map[string]string)
	for _, entry := range environment {
		for index, character := range entry {
			if character == '=' {
				values[entry[:index]] = entry[index+1:]
				break
			}
		}
	}
	if _, ok := values["GOOS"]; ok {
		t.Fatal("GOOS was not removed")
	}
	if values["GOARCH"] != "arm64" {
		t.Fatalf("GOARCH = %q, want arm64", values["GOARCH"])
	}
	if values["TEST_TIP_BUILD"] != "preserved" {
		t.Fatalf("TEST_TIP_BUILD = %q, want preserved", values["TEST_TIP_BUILD"])
	}
}

func TestReportArch(t *testing.T) {
	if got := reportArch(Target{OS: "linux", Arch: "arm"}); got != "arm" {
		t.Fatalf("linux/arm report arch = %q, want arm", got)
	}
	if got := reportArch(Target{OS: "freebsd", Arch: "arm"}); got != "arm" {
		t.Fatalf("freebsd/arm report arch = %q, want arm", got)
	}
	if got := reportArch(Target{OS: "linux", Arch: "armv6l"}); got != "arm" {
		t.Fatalf("linux/armv6l report arch = %q, want arm", got)
	}
}

func TestTargetEnvironment(t *testing.T) {
	t.Setenv("GOAMD64", "v4")
	t.Setenv("GOARM", "7")
	environment := targetEnvironment("goroot", "gocache", Target{OS: "linux", Arch: "arm"})
	values := make(map[string]string)
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[strings.ToUpper(key)] = value
		}
	}
	if values["GOOS"] != "linux" || values["GOARCH"] != "arm" || values["GOARM"] != "6" {
		t.Fatalf("unexpected target environment: GOOS=%q GOARCH=%q GOARM=%q", values["GOOS"], values["GOARCH"], values["GOARM"])
	}
	if values["GOFLAGS"] != releaseGOFLAGS {
		t.Fatalf("GOFLAGS = %q, want %q", values["GOFLAGS"], releaseGOFLAGS)
	}
	if _, ok := values["GOAMD64"]; ok {
		t.Fatal("host GOAMD64 leaked into target environment")
	}
}
