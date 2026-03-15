#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
work_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

portable_root="${work_dir}/ASRSubs-windows-portable"
runtime_source="${work_dir}/python"
mkdir -p "${portable_root}" "${runtime_source}"
touch "${portable_root}/ASRSubs.exe"

cat <<'EOF' > "${runtime_source}/python.exe"
stub
EOF

cat <<'EOF' > "${work_dir}/ffmpeg.exe"
stub
EOF

cat <<'EOF' > "${work_dir}/ffprobe.exe"
stub
EOF

ASRSUBS_PYTHON_STANDALONE="${runtime_source}" \
ASRSUBS_FFMPEG_PATH="${work_dir}/ffmpeg.exe" \
ASRSUBS_FFPROBE_PATH="${work_dir}/ffprobe.exe" \
  "${script_dir}/stage-runtime.sh" windows/amd64 "${portable_root}/ASRSubs.exe" >/dev/null

test -f "${portable_root}/runtime/python/python.exe"
test -f "${portable_root}/runtime/worker.py"
test -f "${portable_root}/runtime/requirements.txt"
test -f "${portable_root}/bin/ffmpeg.exe"
test -f "${portable_root}/bin/ffprobe.exe"

grep -q "ASRSUBS_STAGE_DIR" "${repo_root}/build/windows/installer/project.nsi"
grep -q 'File /r "${ASRSUBS_STAGE_DIR}\\runtime\\\*"' "${repo_root}/build/windows/installer/project.nsi"
grep -q 'File /r "${ASRSUBS_STAGE_DIR}\\bin\\\*"' "${repo_root}/build/windows/installer/project.nsi"
grep -q '!insertmacro wails.files' "${repo_root}/build/windows/installer/project.nsi"

printf 'Windows package layout verification passed.\n'
