package archivecache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Artifact struct {
	Filename string
	Size     int64
	SHA256   string
}

type Result struct {
	Path   string
	Size   int64
	SHA256 string
	Cached bool
}

type Cache struct {
	Dir     string
	BaseURL string
	Client  *http.Client
	Retries int
}

func (cache Cache) Get(ctx context.Context, artifact Artifact) (Result, error) {
	if artifact.Filename == "" || filepath.Base(artifact.Filename) != artifact.Filename {
		return Result{}, fmt.Errorf("invalid archive filename %q", artifact.Filename)
	}
	if artifact.SHA256 != "" {
		digest, err := hex.DecodeString(artifact.SHA256)
		if err != nil || len(digest) != sha256.Size {
			return Result{}, fmt.Errorf("invalid SHA-256 for %s", artifact.Filename)
		}
	}
	if err := os.MkdirAll(cache.Dir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create cache directory: %w", err)
	}

	target := filepath.Join(cache.Dir, artifact.Filename)
	if result, err := verify(target, artifact); err == nil {
		result.Cached = true
		return result, nil
	}

	attempts := cache.Retries
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		result, err := cache.download(ctx, target, artifact)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		if attempt < attempts {
			timer := time.NewTimer(time.Duration(attempt) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return Result{}, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return Result{}, fmt.Errorf("download %s after %d attempt(s): %w", artifact.Filename, attempts, lastErr)
}

func (cache Cache) download(ctx context.Context, target string, artifact Artifact) (Result, error) {
	downloadURL, err := url.JoinPath(cache.BaseURL, artifact.Filename)
	if err != nil {
		return Result{}, fmt.Errorf("build download URL: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return Result{}, fmt.Errorf("create archive request: %w", err)
	}
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("User-Agent", "gosizehistory/1")

	client := cache.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("unexpected response: %s", response.Status)
	}

	temporary, err := os.CreateTemp(cache.Dir, "."+artifact.Filename+".*.part")
	if err != nil {
		return Result{}, fmt.Errorf("create temporary archive: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(temporary, hash), response.Body)
	closeErr := temporary.Close()
	if copyErr != nil {
		return Result{}, fmt.Errorf("write temporary archive: %w", copyErr)
	}
	if closeErr != nil {
		return Result{}, fmt.Errorf("close temporary archive: %w", closeErr)
	}

	digest := hex.EncodeToString(hash.Sum(nil))
	if artifact.Size > 0 && size != artifact.Size {
		return Result{}, fmt.Errorf("size mismatch: got %d, want %d", size, artifact.Size)
	}
	if artifact.SHA256 != "" && !strings.EqualFold(digest, artifact.SHA256) {
		return Result{}, fmt.Errorf("SHA-256 mismatch: got %s, want %s", digest, artifact.SHA256)
	}
	if size == 0 {
		return Result{}, fmt.Errorf("downloaded archive is empty")
	}

	if err := replace(temporaryName, target); err != nil {
		return Result{}, fmt.Errorf("store archive: %w", err)
	}
	return Result{Path: target, Size: size, SHA256: digest}, nil
}

func verify(filename string, artifact Artifact) (Result, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return Result{}, err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return Result{}, fmt.Errorf("cached archive is not a non-empty regular file")
	}
	if artifact.Size > 0 && info.Size() != artifact.Size {
		return Result{}, fmt.Errorf("cached archive size mismatch")
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return Result{}, err
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	if artifact.SHA256 != "" && !strings.EqualFold(digest, artifact.SHA256) {
		return Result{}, fmt.Errorf("cached archive SHA-256 mismatch")
	}
	return Result{Path: filename, Size: info.Size(), SHA256: digest}, nil
}

func replace(source, target string) error {
	if err := os.Rename(source, target); err == nil {
		return nil
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(source, target)
}
