#!/usr/bin/env python3
import json
import os
import re
import sys
import time
from functools import lru_cache


def emit(payload):
    sys.stdout.write(json.dumps(payload))
    sys.stdout.flush()


def fail(command, message, *, exit_code=1):
    emit({"ok": False, "command": command, "error": message})
    return exit_code


def is_test_mode():
    return os.environ.get("ASRSUBS_WORKER_TEST_MODE") == "1"


def validate_path(payload, field_name):
    value = (payload.get(field_name) or "").strip()
    if not value:
        raise ValueError(f"Missing required fields: {field_name}")
    if not os.path.exists(value):
        raise ValueError(f"{field_name} does not exist: {value}")
    return value


def require_transcript(payload):
    transcript = payload.get("transcript") or {}
    text = (transcript.get("text") or "").strip()
    if not text:
        raise ValueError("Missing required fields: transcript")
    return transcript


def split_words(text):
    words = re.findall(r"\S+", text)
    return [{"text": word} for word in words]


def get_field(value, *names):
    for name in names:
        if isinstance(value, dict) and name in value:
            return name, value[name]
        if hasattr(value, name):
            return name, getattr(value, name)
    return None, None


def to_milliseconds(key, value):
    if value is None:
        return None
    if isinstance(value, str):
        value = float(value)
    if key and "ms" in key.lower():
        return int(round(float(value)))
    return int(round(float(value) * 1000))


@lru_cache(maxsize=1)
def load_torch():
    try:
        import torch  # type: ignore
    except ImportError as exc:
        raise RuntimeError("torch is not installed in the managed runtime") from exc
    return torch


def runtime_options():
    torch = load_torch()
    has_cuda = hasattr(torch, "cuda") and torch.cuda.is_available()
    options = {
        "device_map": "cuda:0" if has_cuda else "cpu",
    }
    if has_cuda and hasattr(torch, "bfloat16"):
        options["dtype"] = torch.bfloat16
    elif hasattr(torch, "float32"):
        options["dtype"] = torch.float32
    return options


def instantiate(model_class, model_path):
    options = runtime_options()
    attempts = [
        {"dtype": options.get("dtype"), "device_map": options["device_map"]},
        {"torch_dtype": options.get("dtype"), "device_map": options["device_map"]},
        {"dtype": options.get("dtype")},
        {"torch_dtype": options.get("dtype")},
        {},
    ]
    last_error = None
    for kwargs in attempts:
        kwargs = {key: value for key, value in kwargs.items() if value is not None}
        try:
            return model_class.from_pretrained(model_path, **kwargs)
        except TypeError as exc:
            last_error = exc
            continue
    if last_error is not None:
        raise last_error
    raise RuntimeError("model could not be created")


@lru_cache(maxsize=8)
def load_asr_model(model_path):
    try:
        from qwen_asr import Qwen3ASRModel  # type: ignore
    except ImportError as exc:
        raise RuntimeError("qwen-asr is not installed in the managed runtime") from exc
    return instantiate(Qwen3ASRModel, model_path)


@lru_cache(maxsize=8)
def load_aligner_model(aligner_path):
    try:
        from qwen_asr import Qwen3ForcedAligner  # type: ignore
    except ImportError as exc:
        raise RuntimeError("qwen-asr is not installed in the managed runtime") from exc
    return instantiate(Qwen3ForcedAligner, aligner_path)


def invoke_transcribe(model, audio_path, language):
    attempts = [
        {"audio": audio_path, "language": language},
        {"audio": audio_path},
        {"audio_path": audio_path, "language": language},
        {"audio_path": audio_path},
        {"source": audio_path, "language": language},
        {"source": audio_path},
    ]
    last_error = None
    for kwargs in attempts:
        kwargs = {key: value for key, value in kwargs.items() if value}
        try:
            return model.transcribe(**kwargs)
        except TypeError as exc:
            last_error = exc
            continue
    if last_error is not None:
        raise last_error
    raise RuntimeError("transcribe invocation failed")


def invoke_align(aligner, audio_path, text, language):
    attempts = [
        {"audio": audio_path, "text": text, "language": language},
        {"audio": audio_path, "text": text},
        {"audio_path": audio_path, "text": text, "language": language},
        {"audio_path": audio_path, "text": text},
        {"source": audio_path, "text": text, "language": language},
        {"source": audio_path, "text": text},
    ]
    last_error = None
    for kwargs in attempts:
        kwargs = {key: value for key, value in kwargs.items() if value}
        try:
            return aligner.align(**kwargs)
        except TypeError as exc:
            last_error = exc
            continue
    if last_error is not None:
        raise last_error
    raise RuntimeError("align invocation failed")


def normalize_transcription(result):
    if isinstance(result, list) and result:
        result = result[0]

    _, text = get_field(result, "text", "transcript")
    if text is None:
        raise RuntimeError("qwen transcription result did not include text")

    _, language = get_field(result, "language", "lang")
    _, token_values = get_field(result, "words", "tokens")
    words = []
    if token_values:
        for token in token_values:
            _, token_text = get_field(token, "text", "word", "token")
            if token_text:
                words.append({"text": str(token_text).strip()})
    if not words:
        words = split_words(str(text))

    return {
        "text": str(text).strip(),
        "language": str(language).strip() if language else "",
        "words": words,
    }


def normalize_alignment(result):
    if isinstance(result, list) and result and isinstance(result[0], list):
        result = result[0]
    elif not isinstance(result, list):
        _, nested_words = get_field(result, "words", "items")
        if nested_words is None:
            raise RuntimeError("qwen aligner result did not include words")
        result = nested_words

    aligned_words = []
    for word in result:
        _, text = get_field(word, "text", "word", "token")
        start_key, start_value = get_field(word, "start_time", "start", "startMs")
        end_key, end_value = get_field(word, "end_time", "end", "endMs")
        if text is None or start_value is None or end_value is None:
            continue
        aligned_words.append(
            {
                "text": str(text).strip(),
                "startMs": to_milliseconds(start_key, start_value),
                "endMs": to_milliseconds(end_key, end_value),
                "confidence": 0.99,
            }
        )

    if not aligned_words:
        raise RuntimeError("qwen aligner result did not include usable word timings")

    return {"words": aligned_words}


def handle_transcribe(payload):
    try:
        audio_path = validate_path(payload, "audioPath")
        model_path = validate_path(payload, "modelPath")
        model = load_asr_model(model_path)
        language = (payload.get("language") or "").strip() or None
        result = invoke_transcribe(model, audio_path, language)
        details = normalize_transcription(result)
    except Exception as exc:
        return fail("transcribe", f"Transcription failed: {exc}")

    emit(
        {
            "ok": True,
            "command": "transcribe",
            "message": "Transcription artifacts created.",
            "details": details,
        }
    )
    return 0


def handle_align(payload):
    try:
        audio_path = validate_path(payload, "audioPath")
        validate_path(payload, "modelPath")
        aligner_path = validate_path(payload, "alignerPath")
        transcript = require_transcript(payload)
        text = (transcript.get("text") or "").strip()
        language = (transcript.get("language") or payload.get("language") or "").strip() or None
        aligner = load_aligner_model(aligner_path)
        result = invoke_align(aligner, audio_path, text, language)
        details = normalize_alignment(result)
    except Exception as exc:
        return fail("align", f"Alignment failed: {exc}")

    emit(
        {
            "ok": True,
            "command": "align",
            "message": "Aligned words created.",
            "details": details,
        }
    )
    return 0


def maybe_handle_test_command(command):
    if not is_test_mode():
        return None
    if command == "sleep":
        time.sleep(5)
        emit({"ok": True, "command": "sleep", "message": "test sleep complete"})
        return 0
    if command == "fail":
        sys.stderr.write("worker stderr output\n")
        sys.stderr.flush()
        return fail("fail", "simulated failure")
    return None


def main():
    try:
        payload = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        sys.stderr.write(f"invalid worker request: {exc}\n")
        return fail("unknown", "Worker request was not valid JSON.")

    command = payload.get("command")
    test_exit = maybe_handle_test_command(command)
    if test_exit is not None:
        return test_exit

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

    if command == "align":
        return handle_align(payload)

    return fail(command or "unknown", "Unsupported worker command.")


if __name__ == "__main__":
    raise SystemExit(main())
