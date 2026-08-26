# gosizehistory

`gosizehistory` downloads every binary archive listed by the Go downloads API and writes one JSON document containing the commands and tools shipped for each release and platform.

It inventories regular files directly under:

- `go/bin` (`go`, `gofmt`, and their Windows `.exe` variants)
- `go/pkg/tool/<target>` (`compile`, `link`, `vet`, and the other release-specific tools)

The JSON keeps the path found in the archive, removes `.exe` only from the normalized tool name, and records all sizes in bytes. Architecture names are canonicalized, so API metadata reported as `armv6l` is stored as `arm`; the original archive filename remains unchanged. Source archives and installers are excluded. Stable, beta, and release-candidate versions are included.

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

Use `go run . -h` for all options. Set `-output -` to write JSON to stdout. The filters may be combined; `-version` and `-os` are exact matches, while `-arch arm`, `-arch armv6`, and `-arch armv6l` are aliases. `-latest-per-minor` may be combined with `-os` and `-arch`, but not with `-version`.

## Go tip

Build and measure the current `golang/go` tip for every standalone toolchain platform, replacing any previous tip entry in the existing report:

```console
go run . -tip -tip-workers 2 -output go-tool-sizes.json
```

This mode requires Git, a working bootstrap Go installation, and an existing stable report at the output path. It maintains a dedicated checkout under `.cache/gosizehistory/tip`, force-cleans build products, bootstraps the host toolchain with Go's own make script, and cross-builds every measured binary with `-a`, `CGO_ENABLED=0`, and the exact release flags used by `cmd/dist`: `-trimpath -ldflags=-w -gcflags=cmd/...=-dwarf=false`. The merged report preserves stable releases and contains exactly one tip entry pinned to the Git revision and commit time.

The target matrix comes from `go tool dist list -json`. Android, iOS, JavaScript/Wasm, and WASI/Wasm are excluded because they are execution targets rather than standalone binary toolchain distributions; all other non-broken targets are built. Use `-os` and `-arch` for a focused build, `-tip-ref` to pin another Git ref, or `-bootstrap-goroot` to select the bootstrap toolchain. Use `-tip-base` when the base report differs from the output path.

An interrupted workflow that already produced a standalone tip report can merge it without rebuilding:

```console
go run . -merge-tip-report go-tip-tool-sizes.json -output go-tool-sizes.json
```

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

The static dashboard lives in `docs/` and reads the single versioned snapshot at `docs/data/go-tool-sizes.json`. It charts total executable footprint, binary count, largest binaries, and individual binary evolution, plus CSV and JSON exports. Trend views switch between absolute size, a first-release index, and change from the prior release. A platform-by-release heatmap makes coverage gaps and historical size changes explicit.

The comparison workspace supports release-to-release comparisons for one platform and platform-to-platform comparisons within one release. It includes per-binary additions, removals, diverging size changes, and a ranked change table. The top Platform and Release filters define selected side A; the comparison panel selects side B.

Release comparisons report chronological footprint size change, independent of which selector contains the newer release: negative means the later release is smaller, while positive means it grew. Platform comparisons remain a direct B-vs-A delta.

The dashboard follows the operating-system light or dark preference on first load and persists changes made with the header theme toggle.

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

Refresh the development snapshot independently:

```console
go run . -tip -tip-workers 2 -output docs/data/go-tool-sizes.json
```

Regenerating stable releases preserves an existing embedded tip entry. The archive and source-build caches share the `.cache/gosizehistory` root but use separate subdirectories.

## GitHub Pages

The workflow at `.github/workflows/pages.yml` deploys `docs/` whenever `main` is pushed, and it can also be started manually.

1. Push the repository to GitHub with `main` as its default branch.
2. Open **Settings → Pages** in the GitHub repository.
3. Set **Source** to **GitHub Actions**.
4. Push to `main` or run **Deploy dashboard to GitHub Pages** from the Actions tab.