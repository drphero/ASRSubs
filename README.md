# ASRSubs

ASRSubs is a desktop app for generating subtitles from local media files without sending the audio to a hosted transcription service. You pick a file, choose a local model, let the app transcribe and align the speech, then review and save the resulting `.srt` file.

It is built with Wails and ships with a managed Python runtime plus bundled media tools so the app can prepare audio, run local ASR models, and generate subtitle timing on the machine itself.
<p align="center">
  <img width="400" alt="screenshot" src="https://github.com/user-attachments/assets/16df8d40-3135-410e-a08e-b04d0a405564" />
  <img width="400" alt="SCR-20260421-ctko" src="https://github.com/user-attachments/assets/7189a6d3-7e4c-4400-bcc1-ff99146a3190" />
  <img width="400" alt="SCR-20260421-cvgw" src="https://github.com/user-attachments/assets/62e18060-967f-4e6e-a514-aff501c5ce7c" />

</p>


## What The App Does

- runs speech-to-subtitle workflows locally
- supports both audio and video inputs
- downloads transcription models on demand and reuses them later
- prepares media automatically before transcription
- shows stage-by-stage progress during runtime setup, downloads, transcription, alignment, and subtitle building
- opens the finished subtitle draft in an editor so you can make final fixes before saving

## Typical Workflow

1. Open the app and choose a media file.
2. Prepare the local runtime on the first run.
3. Download the transcription model you want to use.
4. Click **Start Transcription**.
5. Review the generated subtitle draft.
6. Save the final `.srt` file where you want.

## Supported Media

ASRSubs currently accepts:

- `.wav, .mp3, .m4a, .aac, .flac, .ogg, .opus, .mp4, .mov, .m4v, .mkv, .avi, .webm`

## Available Models

The app currently exposes two local ASR model options:

- `Qwen3-ASR-1.7B`
  Best accuracy, slower, larger first download.
- `Qwen3-ASR-0.6B`
  Faster startup, lighter footprint, good for smaller machines and quicker runs.

The app also manages an internal forced aligner automatically when timestamp generation needs it.

## What You Can Adjust

Inside the settings drawer you can:

- switch between the available local models
- download or remove model files
- change subtitle grouping preferences
- change max line length and lines per subtitle
- change the alignment chunk size used for longer files
- re-run local runtime preparation if the managed runtime needs repair

## Project Status

The repository currently builds desktop packages for:

- macOS
- Windows

Windows builds are published as both a portable ZIP and an installer. macOS builds are packaged as a DMG.

## Development

Run the app in development mode:

```bash
wails dev
```

Build the frontend assets directly:

```bash
cd frontend && npm run build
```

## Packaging

### macOS

Build the packaged macOS app and DMG:

```bash
./scripts/build-macos-package.sh
```

That flow:

- builds `ASRSubs.app`
- stages the managed runtime, worker script, requirements, `ffmpeg`, and `ffprobe`
- creates `build/bin/ASRSubs.dmg`

Important staging inputs:

- `ASRSUBS_PYTHON_STANDALONE`
- `ASRSUBS_FFMPEG_PATH`
- `ASRSUBS_FFPROBE_PATH`

If those are not set, the staging helper falls back to packaged runtime/tool locations in the repo or the current shell `PATH`.

### Windows

The repository produces:

- `ASRSubs-<version>-windows-amd64-portable.zip`
- `ASRSubs-<version>-windows-amd64-installer.exe`

The Windows GitHub Actions workflow:

1. installs Go, Node, and Python toolchains
2. installs `ffmpeg` and NSIS
3. builds `ASRSubs.exe`
4. stages the bundled runtime and tools
5. creates the portable ZIP and installer

## Verification Helpers

```bash
./scripts/verify-macos-package.sh
./scripts/verify-windows-package-layout.sh
./scripts/verify-windows-workflow.sh
```

## Notes For Unsigned Builds

- macOS may show a Gatekeeper warning the first time you open a local unsigned build
- Windows may show a SmartScreen warning for unsigned artifacts

For local testing, that is expected.
