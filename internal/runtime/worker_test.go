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

	rootDir := t.TempDir()
	requirementsPath := filepath.Join(rootDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("qwen-asr==0.0.6\n"), 0o644); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	t.Setenv("ASRSUBS_FAKE_PIP_LOG", filepath.Join(rootDir, "pip.log"))

	service := NewServiceAtRoot(
		filepath.Join(rootDir, "managed"),
		WithManagedRuntimeSource(writeFakeRuntimeSource(t)),
		WithRequirementsPath(requirementsPath),
		WithWorkerScriptPath(filepath.Join(rootDir, "worker.py")),
	)

	if _, err := service.EnsureReady(context.Background()); err != nil {
		t.Fatalf("ensure ready: %v", err)
	}

	return service
}

func writeFakeRuntimeSource(t *testing.T) string {
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
		"payload=$(cat)\n" +
		"case \"$payload\" in\n" +
		"  *'\"command\":\"sleep\"'*)\n" +
		"    sleep 5\n" +
		"    ;;\n" +
		"esac\n" +
		"case \"$payload\" in\n" +
		"  *'\"command\":\"fail\"'*)\n" +
		"    echo 'worker stderr output' >&2\n" +
		"    echo '{\"ok\":false,\"command\":\"fail\",\"error\":\"simulated failure\"}'\n" +
		"    exit 1\n" +
		"    ;;\n" +
		"  *'\"command\":\"smoke\"'*)\n" +
		"    echo '{\"ok\":true,\"command\":\"smoke\",\"message\":\"Managed runtime worker is ready.\"}'\n" +
		"    exit 0\n" +
		"    ;;\n" +
		"  *'\"command\":\"transcribe\"'*)\n" +
		"    echo '{\"ok\":true,\"command\":\"transcribe\",\"message\":\"Transcription contract accepted.\",\"details\":{\"stage\":\"queued\"}}'\n" +
		"    exit 0\n" +
		"    ;;\n" +
		"esac\n" +
		"echo '{\"ok\":false,\"command\":\"unknown\",\"error\":\"Unsupported worker command.\"}'\n" +
		"exit 1\n"

	pythonPath := filepath.Join(binDir, "python3")
	if err := os.WriteFile(pythonPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake python: %v", err)
	}

	return rootDir
}
