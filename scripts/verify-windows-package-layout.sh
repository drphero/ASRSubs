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
choco_root="${work_dir}/ProgramData/chocolatey"
mkdir -p "${portable_root}" "${runtime_source}" "${choco_root}/bin" "${choco_root}/lib/ffmpeg/tools/ffmpeg/bin"
touch "${portable_root}/ASRSubs.exe"

cat <<'EOF' > "${runtime_source}/python.exe"
stub
EOF

cat <<'EOF' > "${runtime_source}/pythonw.exe"
stub
EOF

cat <<'EOF' > "${choco_root}/bin/ffmpeg.exe"
shim
EOF

cat <<'EOF' > "${choco_root}/bin/ffprobe.exe"
shim
EOF

cat <<'EOF' > "${choco_root}/lib/ffmpeg/tools/ffmpeg/bin/ffmpeg.exe"
real ffmpeg
EOF

cat <<'EOF' > "${choco_root}/lib/ffmpeg/tools/ffmpeg/bin/ffprobe.exe"
real ffprobe
EOF

ASRSUBS_PYTHON_STANDALONE="${runtime_source}" \
ASRSUBS_FFMPEG_PATH="${choco_root}/bin/ffmpeg.exe" \
ASRSUBS_FFPROBE_PATH="${choco_root}/bin/ffprobe.exe" \
  "${script_dir}/stage-runtime.sh" windows/amd64 "${portable_root}/ASRSubs.exe" >/dev/null

test -f "${portable_root}/runtime/python/python.exe"
test -f "${portable_root}/runtime/python/pythonw.exe"
test -f "${portable_root}/runtime/worker.py"
test -f "${portable_root}/runtime/requirements.txt"
test -f "${portable_root}/bin/ffmpeg.exe"
test -f "${portable_root}/bin/ffprobe.exe"
grep -q "real ffmpeg" "${portable_root}/bin/ffmpeg.exe"
grep -q "real ffprobe" "${portable_root}/bin/ffprobe.exe"

grep -q "ASRSUBS_STAGE_DIR" "${repo_root}/build/windows/installer/project.nsi"
grep -q 'File /r "${ASRSUBS_STAGE_DIR}\\runtime\\\*"' "${repo_root}/build/windows/installer/project.nsi"
grep -q 'File /r "${ASRSUBS_STAGE_DIR}\\bin\\\*"' "${repo_root}/build/windows/installer/project.nsi"
grep -q '!insertmacro wails.files' "${repo_root}/build/windows/installer/project.nsi"

printf 'Windows package layout verification passed.\n'
