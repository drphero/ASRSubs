#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
app_path="${repo_root}/build/bin/ASRSubs.app"
output_path="${repo_root}/build/bin/ASRSubs.dmg"
skip_build="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-build)
      skip_build="true"
      shift
      ;;
    --app)
      app_path="$2"
      shift 2
      ;;
    --output)
      output_path="$2"
      shift 2
      ;;
    *)
      printf 'Unknown argument: %s\n' "$1" >&2
      exit 1
      ;;
  esac
done

if [[ "${skip_build}" != "true" ]]; then
  if [[ "$(uname -s)" != "Darwin" ]]; then
    printf 'macOS packaging must run on macOS.\n' >&2
    exit 1
  fi

  if ! command -v wails >/dev/null 2>&1; then
    printf 'wails CLI is required on PATH to build the macOS package.\n' >&2
    exit 1
  fi

  (cd "${repo_root}" && wails build -clean -platform darwin/universal)
fi

if [[ ! -d "${app_path}" ]]; then
  printf 'App bundle not found at %s\n' "${app_path}" >&2
  exit 1
fi

if ! command -v hdiutil >/dev/null 2>&1; then
  printf 'hdiutil is required to create the DMG.\n' >&2
  exit 1
fi

"${script_dir}/stage-runtime.sh" darwin "${app_path}"

work_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

cp -R "${app_path}" "${work_dir}/ASRSubs.app"
ln -s /Applications "${work_dir}/Applications"
mkdir -p "$(dirname "${output_path}")"
rm -f "${output_path}"

hdiutil create \
  -ov \
  -format UDZO \
  -volname "ASRSubs" \
  -srcfolder "${work_dir}" \
  "${output_path}" >/dev/null

printf 'Created macOS package at %s\n' "${output_path}"
