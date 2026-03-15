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
  "go-version-file: go.mod"
  "cache-dependency-path: frontend/package-lock.json"
  "choco install ffmpeg --version=7.1.1 -y"
  "go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0"
  "wails build -clean -platform windows/amd64 -nsis -webview2 embed"
  "./scripts/stage-runtime.sh windows/amd64"
  "makensis"
  "ASRSubs-windows-portable.zip"
  "ASRSubs-amd64-installer.exe"
  "actions/upload-artifact@v6"
)

for needle in "${checks[@]}"; do
  if ! grep -q "${needle}" "${workflow_path}"; then
    printf 'Workflow check failed: missing "%s"\n' "${needle}" >&2
    exit 1
  fi
done

upload_count="$(grep -c 'actions/upload-artifact@v6' "${workflow_path}")"
if [[ "${upload_count}" -lt 2 ]]; then
  printf 'Workflow check failed: expected two artifact uploads, found %s\n' "${upload_count}" >&2
  exit 1
fi

readme_checks=(
  "macOS"
  "Windows"
  "./scripts/build-macos-package.sh"
  "ASRSubs-windows-portable.zip"
  "ASRSubs-amd64-installer.exe"
  "Gatekeeper"
  "SmartScreen"
)

for needle in "${readme_checks[@]}"; do
  if ! grep -q "${needle}" "${readme_path}"; then
    printf 'README check failed: missing "%s"\n' "${needle}" >&2
    exit 1
  fi
done

printf 'Windows workflow verification passed.\n'
