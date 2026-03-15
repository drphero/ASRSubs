#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
work_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${work_dir}"
}
trap cleanup EXIT

app_path="${work_dir}/build/bin/ASRSubs.app"
mkdir -p "${app_path}/Contents/MacOS"
touch "${app_path}/Contents/MacOS/ASRSubs"

runtime_source="${work_dir}/sources/python"
mkdir -p "${runtime_source}/bin"
cat <<'EOF' > "${runtime_source}/bin/python3"
#!/bin/sh
exit 0
EOF
chmod +x "${runtime_source}/bin/python3"

cat <<'EOF' > "${work_dir}/ffmpeg"
#!/bin/sh
exit 0
EOF
chmod +x "${work_dir}/ffmpeg"

cat <<'EOF' > "${work_dir}/ffprobe"
#!/bin/sh
exit 0
EOF
chmod +x "${work_dir}/ffprobe"

ASRSUBS_PYTHON_STANDALONE="${runtime_source}" \
ASRSUBS_FFMPEG_PATH="${work_dir}/ffmpeg" \
ASRSUBS_FFPROBE_PATH="${work_dir}/ffprobe" \
  "${script_dir}/stage-runtime.sh" darwin "${app_path}" >/dev/null

test -f "${app_path}/Contents/Resources/runtime/worker.py"
test -f "${app_path}/Contents/Resources/runtime/requirements.txt"
test -f "${app_path}/Contents/Resources/runtime/python/bin/python3"
test -f "${app_path}/Contents/Resources/bin/ffmpeg"
test -f "${app_path}/Contents/Resources/bin/ffprobe"

if command -v hdiutil >/dev/null 2>&1; then
  output_path="${work_dir}/build/bin/ASRSubs.dmg"
  ASRSUBS_PYTHON_STANDALONE="${runtime_source}" \
  ASRSUBS_FFMPEG_PATH="${work_dir}/ffmpeg" \
  ASRSUBS_FFPROBE_PATH="${work_dir}/ffprobe" \
    "${script_dir}/build-macos-package.sh" --skip-build --app "${app_path}" --output "${output_path}" >/dev/null
  test -f "${output_path}"
fi

printf 'macOS packaging verification passed.\n'
