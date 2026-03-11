#!/usr/bin/env python3
import json
import sys


def emit(payload):
    sys.stdout.write(json.dumps(payload))
    sys.stdout.flush()


def fail(command, message, *, exit_code=1):
    emit({"ok": False, "command": command, "error": message})
    return exit_code


def handle_transcribe(payload):
    missing = []
    if not payload.get("audioPath"):
        missing.append("audioPath")
    if not payload.get("modelPath"):
        missing.append("modelPath")

    if missing:
        return fail("transcribe", f"Missing required fields: {', '.join(missing)}")

    emit(
        {
            "ok": True,
            "command": "transcribe",
            "message": "Transcription contract accepted.",
            "details": {
                "audioPath": payload["audioPath"],
                "modelPath": payload["modelPath"],
                "language": payload.get("language") or "auto",
            },
        }
    )
    return 0


def main():
    try:
        payload = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        sys.stderr.write(f"invalid worker request: {exc}\n")
        return fail("unknown", "Worker request was not valid JSON.")

    command = payload.get("command")
    if command == "smoke":
        emit(
            {
                "ok": True,
                "command": "smoke",
                "message": "Managed runtime worker is ready.",
            }
        )
        return 0

    if command == "transcribe":
        return handle_transcribe(payload)

    return fail(command or "unknown", "Unsupported worker command.")


if __name__ == "__main__":
    raise SystemExit(main())
