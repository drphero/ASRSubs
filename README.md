# ASRSubs

ASRSubs is a Wails desktop app for local subtitle generation. Phase 5 packages the app as a macOS DMG and adds a Windows GitHub Actions pipeline that publishes both a portable bundle and an installer.

## Development

Run the live app shell with:

```bash
wails dev
```

Build the production frontend assets with:

```bash
cd frontend && npm run build
```

## Packaging

### macOS

Package the signed-off macOS delivery build with:

```bash
./scripts/build-macos-package.sh
```

That flow:

- builds `ASRSubs.app` with Wails
- stages the managed Python runtime, `worker.py`, `requirements.txt`, `ffmpeg`, and `ffprobe` into `ASRSubs.app/Contents/Resources`
- creates `build/bin/ASRSubs.dmg` with a drag-to-Applications layout

Inputs for staging:

- set `ASRSUBS_PYTHON_STANDALONE` to a standalone Python directory for the packaged runtime
- set `ASRSUBS_FFMPEG_PATH` and `ASRSUBS_FFPROBE_PATH` when you want to pin specific macOS binaries
- if those env vars are omitted, the staging helper falls back to `packaging/runtime/darwin/python`, `packaging/tools/darwin/ffmpeg`, `packaging/tools/darwin/ffprobe`, or the current shell `PATH`

Unsigned macOS builds will trigger a Gatekeeper warning on first open. For local testing, open the app from Finder, then approve it from `System Settings > Privacy & Security` if macOS blocks the first launch.

### Windows

The repository publishes two Windows deliverables:

- `ASRSubs-<version>-windows-amd64-portable.zip`: the app executable plus bundled runtime and `ffmpeg` payload
- `ASRSubs-<version>-windows-amd64-installer.exe`: the NSIS installer with the same staged runtime tree

The workflow derives `<version>` from `info.productVersion` in `wails.json`, so the artifact filenames match the embedded app version metadata.

The GitHub Actions workflow lives at `.github/workflows/build-windows.yml`. It:

1. restores Go, Node, and Python toolchains
2. installs `ffmpeg`
3. builds `ASRSubs.exe` with `wails build -clean -platform windows/amd64 -nsis -webview2 embed`
4. stages the portable runtime layout with `./scripts/stage-runtime.sh windows/amd64`
5. creates the portable ZIP and a custom NSIS installer that copies the staged `runtime/` and `bin/` directories

Local verification helpers:

```bash
./scripts/verify-macos-package.sh
./scripts/verify-windows-package-layout.sh
./scripts/verify-windows-workflow.sh
```

Unsigned Windows builds will surface a SmartScreen warning. For manual testing, use `More info` then `Run anyway` once you trust the artifact source.
