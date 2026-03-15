#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
workflow_path="${repo_root}/.github/workflows/build-windows.yml"
readme_path="${repo_root}/README.md"

if [[ ! -f "${workflow_path}" ]]; then
  printf 'Workflow file is missing: %s\n' "${workflow_path}" >&2
  exit 1
fi

if [[ ! -f "${readme_path}" ]]; then
  printf 'README is missing: %s\n' "${readme_path}" >&2
  exit 1
fi

checks=(
  "runs-on: windows-latest"
  "actions/checkout@v6"
  "actions/setup-go@v6"
  "actions/setup-node@v6"
  "actions/setup-python@v6"
  "Get-Content \"wails.json\" | ConvertFrom-Json"
  "ASRSUBS_VERSION="
  "ASRSUBS_PORTABLE_ZIP=ASRSubs-\$version-windows-amd64-portable.zip"
  "ASRSUBS_INSTALLER_EXE=ASRSubs-\$version-windows-amd64-installer.exe"
  "go-version-file: go.mod"
  "cache-dependency-path: frontend/package-lock.json"
  "choco install ffmpeg --version=7.1.1 -y"
  "choco install nsis -y"
  "go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0"
  "wails build -clean -platform windows/amd64 -nsis -webview2 embed"
  "./scripts/stage-runtime.sh windows/amd64"
  "/c/ProgramData/chocolatey/lib/ffmpeg/tools/ffmpeg/bin/ffmpeg.exe"
  "/c/ProgramData/chocolatey/lib/ffmpeg/tools/ffmpeg/bin/ffprobe.exe"
  "Smoke-test staged ffmpeg tools"
  "Resolve-Path \"build/bin/ASRSubs-windows-portable/runtime/python/pythonw.exe\""
  "Resolve-Path \"build/bin/ASRSubs-windows-portable/bin/ffmpeg.exe\""
  "Resolve-Path \"build/bin/ASRSubs-windows-portable/bin/ffprobe.exe\""
  "throw \"staged ffmpeg.exe failed to execute\""
  "throw \"staged ffprobe.exe failed to execute\""
  "Get-Command makensis -ErrorAction SilentlyContinue"
  "ProgramData\\chocolatey\\bin\\makensis.exe"
  "throw \"makensis executable not found after NSIS install\""
  "Resolve-Path \"build/bin/ASRSubs.exe\""
  "Resolve-Path \"build/bin/ASRSubs-windows-portable\""
  "Push-Location \"build/windows/installer\""
  "\"/DARG_WAILS_AMD64_BINARY=\$amd64Binary\""
  "\"/DASRSUBS_STAGE_DIR=\$stageDir\""
  "\"project.nsi\""
  "build/bin/\$env:ASRSUBS_PORTABLE_ZIP"
  "Move-Item \"build/bin/ASRSubs-amd64-installer.exe\" \"build/bin/\$env:ASRSUBS_INSTALLER_EXE\" -Force"
  "name: \${{ env.ASRSUBS_PORTABLE_ZIP }}"
  "path: build/bin/\${{ env.ASRSUBS_PORTABLE_ZIP }}"
  "name: \${{ env.ASRSUBS_INSTALLER_EXE }}"
  "path: build/bin/\${{ env.ASRSUBS_INSTALLER_EXE }}"
  "actions/upload-artifact@v6"
)

for needle in "${checks[@]}"; do
  if ! grep -Fq "${needle}" "${workflow_path}"; then
    printf 'Workflow check failed: missing "%s"\n' "${needle}" >&2
    exit 1
  fi
done

for forbidden in \
  "/c/ProgramData/chocolatey/bin/ffmpeg.exe" \
  "/c/ProgramData/chocolatey/bin/ffprobe.exe"; do
  if grep -Fq "${forbidden}" "${workflow_path}"; then
    printf 'Workflow check failed: unexpected Chocolatey shim path "%s"\n' "${forbidden}" >&2
    exit 1
  fi
done

upload_count="$(grep -Fc 'actions/upload-artifact@v6' "${workflow_path}")"
if [[ "${upload_count}" -lt 2 ]]; then
  printf 'Workflow check failed: expected two artifact uploads, found %s\n' "${upload_count}" >&2
  exit 1
fi

readme_checks=(
  "macOS"
  "Windows"
  "./scripts/build-macos-package.sh"
  "ASRSubs-<version>-windows-amd64-portable.zip"
  "ASRSubs-<version>-windows-amd64-installer.exe"
  "wails.json"
  "Gatekeeper"
  "SmartScreen"
)

for needle in "${readme_checks[@]}"; do
  if ! grep -Fq "${needle}" "${readme_path}"; then
    printf 'README check failed: missing "%s"\n' "${needle}" >&2
    exit 1
  fi
done

printf 'Windows workflow verification passed.\n'
