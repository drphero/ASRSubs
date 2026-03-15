package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkerRunsSmokeCommand(t *testing.T) {
	service := newPreparedRuntimeService(t)

	response, err := service.Smoke(context.Background())
	if err != nil {
		t.Fatalf("smoke worker: %v", err)
	}

	if !response.OK {
		t.Fatal("expected smoke response to succeed")
	}

	if response.Command != "smoke" {
		t.Fatalf("unexpected command: %s", response.Command)
	}
}

func TestWorkerReturnsStructuredAlignmentDetails(t *testing.T) {
	service := newPreparedRuntimeService(t)
	audioPath := writeFakeAudioFile(t, "clip.wav")
	modelPath := writeFakeModelDir(t, "asr")
	alignerPath := writeFakeModelDir(t, "aligner")

	response, err := service.RunWorker(context.Background(), WorkerRequest{
		Command:     "align",
		AudioPath:   audioPath,
		ModelPath:   modelPath,
		AlignerPath: alignerPath,
		Transcript: &TranscriptPayload{
			Text:  "alpha beta",
			Words: []TranscriptToken{{Text: "alpha"}, {Text: "beta"}},
		},
	})
	if err != nil {
		t.Fatalf("align worker: %v", err)
	}

	var payload AlignmentPayload
	if err := response.DecodeDetails(&payload); err != nil {
		t.Fatalf("decode details: %v", err)
	}
	if len(payload.Words) != 2 {
		t.Fatalf("expected 2 aligned words, got %d", len(payload.Words))
	}
}

func TestWorkerReturnsStructuredAlignmentDetailsFromForcedAlignResultItems(t *testing.T) {
	service := newPreparedRuntimeServiceWithAlignmentShape(t, "forced_align_result")
	audioPath := writeFakeAudioFile(t, "clip.wav")
	modelPath := writeFakeModelDir(t, "asr")
	alignerPath := writeFakeModelDir(t, "aligner")

	response, err := service.RunWorker(context.Background(), WorkerRequest{
		Command:     "align",
		AudioPath:   audioPath,
		ModelPath:   modelPath,
		AlignerPath: alignerPath,
		Transcript: &TranscriptPayload{
			Text:  "alpha beta",
			Words: []TranscriptToken{{Text: "alpha"}, {Text: "beta"}},
		},
	})
	if err != nil {
		t.Fatalf("align worker: %v", err)
	}

	var payload AlignmentPayload
	if err := response.DecodeDetails(&payload); err != nil {
		t.Fatalf("decode details: %v", err)
	}
	if len(payload.Words) != 2 {
		t.Fatalf("expected 2 aligned words, got %d", len(payload.Words))
	}
}

func TestWorkerReturnsRealModelTranscriptionPayload(t *testing.T) {
	service := newPreparedRuntimeService(t)
	audioPath := writeFakeAudioFile(t, "my_demo_file.wav")
	modelPath := writeFakeModelDir(t, "asr")

	response, err := service.RunWorker(context.Background(), WorkerRequest{
		Command:   "transcribe",
		AudioPath: audioPath,
		ModelPath: modelPath,
	})
	if err != nil {
		t.Fatalf("transcribe worker: %v", err)
	}

	var payload TranscriptPayload
	if err := response.DecodeDetails(&payload); err != nil {
		t.Fatalf("decode details: %v", err)
	}
	if payload.Text != "real model transcript" {
		t.Fatalf("expected fake qwen result, got %q", payload.Text)
	}
	if payload.Text == "my demo file" {
		t.Fatal("worker derived transcript from the filename instead of the model output")
	}
	if payload.Language != "en" {
		t.Fatalf("expected detected language, got %q", payload.Language)
	}
}

func TestWorkerSourceNoLongerContainsFilenameFallback(t *testing.T) {
	workerPath, err := filepath.Abs(filepath.Join("worker.py"))
	if err != nil {
		t.Fatalf("resolve worker path: %v", err)
	}

	data, err := os.ReadFile(workerPath)
	if err != nil {
		t.Fatalf("read worker source: %v", err)
	}

	source := string(data)
	for _, banned := range []string{"sample transcript", "def derive_text", "def tokenize"} {
		if strings.Contains(source, banned) {
			t.Fatalf("worker source still contains placeholder logic: %s", banned)
		}
	}
}

func TestWorkerSurfacesStructuredFailure(t *testing.T) {
	service := newPreparedRuntimeService(t)

	_, err := service.RunWorker(context.Background(), WorkerRequest{Command: "fail"})
	if err == nil {
		t.Fatal("expected worker failure")
	}

	workerErr, ok := err.(*WorkerError)
	if !ok {
		t.Fatalf("expected worker error, got %T", err)
	}

	if workerErr.Message != "simulated failure" {
		t.Fatalf("unexpected error message: %s", workerErr.Message)
	}

	if !strings.Contains(workerErr.Stderr, "worker stderr output") {
		t.Fatalf("expected stderr to be captured, got %q", workerErr.Stderr)
	}
}

func TestWorkerCancelsWithContext(t *testing.T) {
	service := newPreparedRuntimeService(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := service.RunWorker(ctx, WorkerRequest{Command: "sleep"})
	if err == nil {
		t.Fatal("expected worker cancellation")
	}

	workerErr, ok := err.(*WorkerError)
	if !ok {
		t.Fatalf("expected worker error, got %T", err)
	}

	if workerErr.Message != "worker canceled" {
		t.Fatalf("unexpected cancellation message: %s", workerErr.Message)
	}
}

func newPreparedRuntimeService(t *testing.T) *Service {
	t.Helper()

	return newPreparedRuntimeServiceWithAlignmentShape(t, "nested_list")
}

func newPreparedRuntimeServiceWithAlignmentShape(t *testing.T, alignmentShape string) *Service {
	t.Helper()

	rootDir := t.TempDir()
	requirementsPath := filepath.Join(rootDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("qwen-asr==0.0.6\n"), 0o644); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	t.Setenv("ASRSUBS_FAKE_PIP_LOG", filepath.Join(rootDir, "pip.log"))
	fakeModuleRoot := writeFakeQwenModule(t, alignmentShape)

	service := NewServiceAtRoot(
		filepath.Join(rootDir, "managed"),
		WithManagedRuntimeSource(writeFakeRuntimeSource(t, fakeModuleRoot)),
		WithRequirementsPath(requirementsPath),
		WithWorkerScriptPath(mustWorkerPath(t)),
	)

	if _, err := service.EnsureReady(context.Background()); err != nil {
		t.Fatalf("ensure ready: %v", err)
	}

	return service
}

func writeFakeRuntimeSource(t *testing.T, fakeModuleRoot string) string {
	t.Helper()

	rootDir := t.TempDir()
	binDir := filepath.Join(rootDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create runtime bin dir: %v", err)
	}

	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-m\" ] && [ \"$2\" = \"pip\" ]; then\n" +
		"  if [ -n \"$ASRSUBS_FAKE_PIP_LOG\" ]; then\n" +
		"    echo \"$@\" >> \"$ASRSUBS_FAKE_PIP_LOG\"\n" +
		"  fi\n" +
		"  exit 0\n" +
		"fi\n" +
		"export ASRSUBS_WORKER_TEST_MODE=1\n" +
		"export PYTHONPATH=\"" + fakeModuleRoot + "${PYTHONPATH:+:$PYTHONPATH}\"\n" +
		"exec python3 \"$@\"\n"

	pythonPath := filepath.Join(binDir, "python3")
	if err := os.WriteFile(pythonPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake python: %v", err)
	}

	return rootDir
}

func writeFakeQwenModule(t *testing.T, alignmentShape string) string {
	t.Helper()

	rootDir := t.TempDir()
	moduleDir := filepath.Join(rootDir, "qwen_asr")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("create fake qwen module: %v", err)
	}

	fakeQwen := "import os\n" +
		"\n" +
		"class _Result:\n" +
		"    def __init__(self, text, language='en'):\n" +
		"        self.text = text\n" +
		"        self.language = language\n" +
		"\n" +
		"class _Word:\n" +
		"    def __init__(self, text, start_time, end_time):\n" +
		"        self.text = text\n" +
		"        self.start_time = start_time\n" +
		"        self.end_time = end_time\n" +
		"\n" +
		"class ForcedAlignResult:\n" +
		"    def __init__(self, items):\n" +
		"        self.items = items\n" +
		"\n" +
		"class Qwen3ASRModel:\n" +
		"    def __init__(self, model_path, **kwargs):\n" +
		"        self.model_path = model_path\n" +
		"\n" +
		"    @classmethod\n" +
		"    def from_pretrained(cls, model_path, **kwargs):\n" +
		"        return cls(model_path, **kwargs)\n" +
		"\n" +
		"    def transcribe(self, audio=None, audio_path=None, source=None, language=None, **kwargs):\n" +
		"        selected = audio or audio_path or source\n" +
		"        if not selected:\n" +
		"            raise RuntimeError('audio path missing')\n" +
		"        return [_Result('real model transcript', language or 'en')]\n" +
		"\n" +
		"class Qwen3ForcedAligner:\n" +
		"    def __init__(self, model_path, **kwargs):\n" +
		"        self.model_path = model_path\n" +
		"\n" +
		"    @classmethod\n" +
		"    def from_pretrained(cls, model_path, **kwargs):\n" +
		"        return cls(model_path, **kwargs)\n" +
		"\n" +
		"    def align(self, audio=None, audio_path=None, source=None, text=None, language=None, **kwargs):\n" +
		"        if not (audio or audio_path or source):\n" +
		"            raise RuntimeError('audio path missing')\n" +
		"        words = []\n" +
		"        for index, token in enumerate((text or '').split()):\n" +
		"            start = index * 0.4\n" +
		"            words.append(_Word(token, start, start + 0.24))\n" +
		"        if os.environ.get(\"ASRSUBS_TEST_ALIGNMENT_SHAPE\") == \"forced_align_result\":\n" +
		"            return [ForcedAlignResult(words)]\n" +
		"        return [words]\n"
	if err := os.WriteFile(filepath.Join(moduleDir, "__init__.py"), []byte(fakeQwen), 0o644); err != nil {
		t.Fatalf("write fake qwen module: %v", err)
	}

	fakeTorch := "float32 = 'float32'\n" +
		"bfloat16 = 'bfloat16'\n" +
		"class cuda:\n" +
		"    @staticmethod\n" +
		"    def is_available():\n" +
		"        return False\n"
	if err := os.WriteFile(filepath.Join(rootDir, "torch.py"), []byte(fakeTorch), 0o644); err != nil {
		t.Fatalf("write fake torch module: %v", err)
	}

	if alignmentShape != "" {
		t.Setenv("ASRSUBS_TEST_ALIGNMENT_SHAPE", alignmentShape)
	}

	return rootDir
}

func writeFakeAudioFile(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("audio"), 0o644); err != nil {
		t.Fatalf("write fake audio: %v", err)
	}
	return path
}

func writeFakeModelDir(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create fake model dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "weights.bin"), []byte(name), 0o644); err != nil {
		t.Fatalf("write fake model weights: %v", err)
	}
	return path
}

func mustWorkerPath(t *testing.T) string {
	t.Helper()

	workerPath, err := filepath.Abs(filepath.Join("worker.py"))
	if err != nil {
		t.Fatalf("resolve worker path: %v", err)
	}
	return workerPath
}

func TestWorkerDecodeDetailsRejectsEmptyPayload(t *testing.T) {
	err := (WorkerResponse{}).DecodeDetails(&TranscriptPayload{})
	if err == nil {
		t.Fatal("expected empty details to fail")
	}
}
