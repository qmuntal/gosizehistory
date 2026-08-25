package goreleases

import (
	"context"
	"encoding/json"
	"fmt"
	goversion "go/version"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const DefaultMetadataURL = "https://go.dev/dl/?mode=json&include=all"

type Release struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Files   []File `json:"files"`
}

type File struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
}

type Filter struct {
	Version        string
	OS             string
	Arch           string
	LatestPerMinor bool
}

type Archive struct {
	Release string
	Stable  bool
	File    File
}

func Load(ctx context.Context, client *http.Client, source string) ([]Release, error) {
	reader, closeReader, err := open(ctx, client, source)
	if err != nil {
		return nil, err
	}
	defer closeReader()

	var releases []Release
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode release metadata: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("release metadata is empty")
	}
	return releases, nil
}

func SelectArchives(releases []Release, filter Filter) []Archive {
	var latestByMinor map[string]string
	if filter.LatestPerMinor {
		latestByMinor = latestStableVersions(releases)
	}

	var archives []Archive
	for _, release := range releases {
		if filter.Version != "" && release.Version != filter.Version {
			continue
		}
		if filter.LatestPerMinor {
			minor, ok := minorVersion(release.Version)
			if !ok || latestByMinor[minor] != release.Version {
				continue
			}
		}
		for _, file := range release.Files {
			if file.Kind != "archive" {
				continue
			}
			if filter.OS != "" && file.OS != filter.OS {
				continue
			}
			if filter.Arch != "" && file.Arch != filter.Arch {
				continue
			}
			archives = append(archives, Archive{Release: release.Version, Stable: release.Stable, File: file})
		}
	}
	return archives
}

func latestStableVersions(releases []Release) map[string]string {
	latest := make(map[string]string)
	for _, release := range releases {
		if !release.Stable {
			continue
		}
		minor, ok := minorVersion(release.Version)
		if !ok {
			continue
		}
		if current := latest[minor]; current == "" || goversion.Compare(release.Version, current) > 0 {
			latest[minor] = release.Version
		}
	}
	return latest
}

func minorVersion(version string) (string, bool) {
	if !goversion.IsValid(version) {
		return "", false
	}
	languageVersion := goversion.Lang(version)
	if !strings.Contains(strings.TrimPrefix(languageVersion, "go"), ".") {
		return "", false
	}
	return languageVersion, true
}

func open(ctx context.Context, client *http.Client, source string) (io.Reader, func(), error) {
	parsedURL, err := url.Parse(source)
	if err == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https") {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("create metadata request: %w", err)
		}
		request.Header.Set("User-Agent", "gosizehistory/1")

		response, err := client.Do(request)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch release metadata: %w", err)
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			return nil, nil, fmt.Errorf("fetch release metadata: %s", response.Status)
		}
		return response.Body, func() { response.Body.Close() }, nil
	}

	file, err := os.Open(source)
	if err != nil {
		return nil, nil, fmt.Errorf("open release metadata: %w", err)
	}
	return file, func() { file.Close() }, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode release metadata: multiple JSON values")
		}
		return fmt.Errorf("decode release metadata: %w", err)
	}
	return nil
}
