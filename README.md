# gosizehistory

`gosizehistory` downloads every binary archive listed by the Go downloads API and writes one JSON document containing the commands and tools shipped for each release and platform.

It inventories regular files directly under:

- `go/bin` (`go`, `gofmt`, and their Windows `.exe` variants)
- `go/pkg/tool/<target>` (`compile`, `link`, `vet`, and the other release-specific tools)

The JSON keeps the path found in the archive, removes `.exe` only from the normalized tool name, and records all sizes in bytes. Source archives and installers are excluded. Stable, beta, and release-candidate versions are included.

## Build and test

```console
go test ./...
go build -o gosizehistory .
```

On Windows, the built executable is `gosizehistory.exe`.

## Usage

Preview the current workload without downloading archives:

```console
go run . -dry-run
```

An unfiltered run is intentionally large. At the time this project was created, the API described about 530 GiB across more than 6,700 archives. The total grows as Go releases are added.

Run a small slice first:

```console
go run . -version go1.27.0 -os linux -arch amd64 -output go-tool-sizes.json
```

Collect only the latest stable patch release from every available Go minor line:

```console
go run . -latest-per-minor -workers 4 -output go-tool-sizes.json
```

Run the complete history:

```console
go run . -workers 4 -output go-tool-sizes.json
```

Archives are retained under `.cache/gosizehistory` by default. Completed files are verified and reused, so restarting the command resumes the collection. Downloads and JSON output use temporary files followed by a rename, preventing interrupted writes from appearing complete.

The metadata can also come from a saved API response such as the existing `dl.json`:

```console
go run . -metadata dl.json -dry-run
```

Use `go run . -h` for all options. Set `-output -` to write JSON to stdout. The `-version`, `-os`, and `-arch` filters are exact matches and may be combined. `-latest-per-minor` may be combined with `-os` and `-arch`, but not with `-version`.

## Output

```json
{
  "schema_version": 1,
  "generated_at": "2026-08-25T12:00:00Z",
  "source": "https://go.dev/dl/?mode=json&include=all",
  "releases": [
    {
      "version": "go1.27.0",
      "stable": true,
      "platforms": [
        {
          "os": "linux",
          "arch": "amd64",
          "archive": {
            "filename": "go1.27.0.linux-amd64.tar.gz",
            "size": 70400000,
            "sha256": "..."
          },
          "tools": [
            {
              "name": "go",
              "path": "go/bin/go",
              "category": "command",
              "size": 18500000
            },
            {
              "name": "compile",
              "path": "go/pkg/tool/linux_amd64/compile",
              "category": "tool",
              "size": 31000000
            }
          ]
        }
      ]
    }
  ]
}
```

Each API archive remains a separate platform record. This matters for historical releases that published multiple archives for the same OS and architecture. For legacy metadata with a missing size or checksum, the report records the values measured from the downloaded archive.

## Dashboard

The static dashboard lives in `docs/` and reads the versioned snapshot at `docs/data/go-tool-sizes.json`. It charts total executable footprint, binary count, largest binaries, and individual binary evolution, plus CSV and JSON exports. Its comparison workspace supports release-to-release comparisons for one platform and platform-to-platform comparisons within one release, including per-binary additions, removals, and size deltas. The top Platform and Snapshot filters define comparison side A; the comparison panel selects only side B.

The header uses the unmodified Go gopher by Renée French, licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).

Preview it locally from the repository root:

```console
python -m http.server 4173 --directory docs
```

Then open <http://localhost:4173/>.

Refresh the published dataset after collecting new releases:

```console
go run . -latest-per-minor -workers 4 -output docs/data/go-tool-sizes.json
```

The archive cache is shared with normal collector runs, so an update downloads only archives that are not already present.

## GitHub Pages

The workflow at `.github/workflows/pages.yml` deploys `docs/` whenever `main` is pushed, and it can also be started manually.

1. Push the repository to GitHub with `main` as its default branch.
2. Open **Settings → Pages** in the GitHub repository.
3. Set **Source** to **GitHub Actions**.
4. Push to `main` or run **Deploy dashboard to GitHub Pages** from the Actions tab.