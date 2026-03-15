#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./scripts/stage-runtime.sh <platform> <target>

Examples:
  ./scripts/stage-runtime.sh darwin /path/to/ASRSubs.app
  ./scripts/stage-runtime.sh windows/amd64 build/bin/ASRSubs-windows-portable/ASRSubs.exe
EOF
}

if [[ $# -lt 2 ]]; then
  usage >&2
  exit 1
fi

platform="$1"
target="$2"
platform_family="${platform%%/*}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

resolve_runtime_source() {
  local candidate="${ASRSUBS_PYTHON_STANDALONE:-${repo_root}/packaging/runtime/${platform_family}/python}"
  if [[ ! -d "${candidate}" ]]; then
    printf 'Managed runtime source not found: %s\n' "${candidate}" >&2
    printf 'Set ASRSUBS_PYTHON_STANDALONE to a standalone Python directory for %s packaging.\n' "${platform_family}" >&2
    exit 1
  fi
  printf '%s\n' "${candidate}"
}

resolve_binary_source() {
  local env_var="$1"
  local default_name="$2"
  local candidate="${!env_var:-}"
  if [[ -n "${candidate}" && -f "${candidate}" ]]; then
    printf '%s\n' "${candidate}"
    return 0
  fi

  local packaged_candidate="${repo_root}/packaging/tools/${platform_family}/${default_name}"
  if [[ -f "${packaged_candidate}" ]]; then
    printf '%s\n' "${packaged_candidate}"
    return 0
  fi

  local command_name="${default_name%.*}"
  if command -v "${command_name}" >/dev/null 2>&1; then
    command -v "${command_name}"
    return 0
  fi

  printf 'Required binary %s is unavailable. Set %s or add it under packaging/tools/%s/.\n' "${default_name}" "${env_var}" "${platform_family}" >&2
  exit 1
}

stage_root_for_target() {
  local candidate="$1"

  if [[ "${platform_family}" == "darwin" ]]; then
    if [[ -d "${candidate}" && "${candidate}" == *.app ]]; then
      printf '%s\n' "${candidate}/Contents/Resources"
      return 0
    fi

    printf 'darwin staging target must be an .app bundle path.\n' >&2
    exit 1
  fi

  if [[ "${candidate}" == *.exe ]]; then
    printf '%s\n' "$(dirname "${candidate}")"
    return 0
  fi

  printf '%s\n' "${candidate}"
}

copy_tree_contents() {
  local source_dir="$1"
  local destination_dir="$2"

  rm -rf "${destination_dir}"
  mkdir -p "${destination_dir}"
  cp -R "${source_dir}/." "${destination_dir}/"
}

copy_file() {
  local source_file="$1"
  local destination_file="$2"

  mkdir -p "$(dirname "${destination_file}")"
  cp "${source_file}" "${destination_file}"
}

runtime_source="$(resolve_runtime_source)"
worker_source="${repo_root}/internal/runtime/worker.py"
requirements_source="${repo_root}/internal/runtime/requirements.txt"

ffmpeg_name="ffmpeg"
ffprobe_name="ffprobe"
if [[ "${platform_family}" == "windows" ]]; then
  ffmpeg_name="ffmpeg.exe"
  ffprobe_name="ffprobe.exe"
fi

ffmpeg_source="$(resolve_binary_source ASRSUBS_FFMPEG_PATH "${ffmpeg_name}")"
ffprobe_source="$(resolve_binary_source ASRSUBS_FFPROBE_PATH "${ffprobe_name}")"
stage_root="$(stage_root_for_target "${target}")"

mkdir -p "${stage_root}/runtime" "${stage_root}/bin"
copy_tree_contents "${runtime_source}" "${stage_root}/runtime/python"
copy_file "${worker_source}" "${stage_root}/runtime/worker.py"
copy_file "${requirements_source}" "${stage_root}/runtime/requirements.txt"
copy_file "${ffmpeg_source}" "${stage_root}/bin/${ffmpeg_name}"
copy_file "${ffprobe_source}" "${stage_root}/bin/${ffprobe_name}"
chmod +x "${stage_root}/bin/${ffmpeg_name}" "${stage_root}/bin/${ffprobe_name}"

if [[ "${platform_family}" != "windows" ]]; then
  chmod +x "${stage_root}/runtime/python/bin/python3" || true
fi

printf 'Staged runtime assets into %s\n' "${stage_root}"
