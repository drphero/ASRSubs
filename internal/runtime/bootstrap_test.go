package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeBootstrapCreatesManagedRuntime(t *testing.T) {
	rootDir := t.TempDir()
	requirementsPath := filepath.Join(rootDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("qwen-asr==0.0.6\n"), 0o644); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	logPath := filepath.Join(rootDir, "pip.log")
	t.Setenv("ASRSUBS_FAKE_PIP_LOG", logPath)

	service := NewServiceAtRoot(
		filepath.Join(rootDir, "managed"),
		WithManagedRuntimeSource(writeFakeRuntimeSource(t, writeFakeQwenModule(t, "nested_list"))),
		WithRequirementsPath(requirementsPath),
		WithWorkerScriptPath(filepath.Join(rootDir, "worker.py")),
	)

	status, err := service.EnsureReady(context.Background())
	if err != nil {
		t.Fatalf("ensure ready: %v", err)
	}

	if status.State != "ready" {
		t.Fatalf("expected ready state, got %s", status.State)
	}

	if !fileExists(service.PythonPath()) {
		t.Fatalf("expected managed python at %s", service.PythonPath())
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read pip log: %v", err)
	}

	if !strings.Contains(string(logData), "install --disable-pip-version-check -r") {
		t.Fatalf("expected pip install to run, got %s", string(logData))
	}
}

func TestRuntimeBootstrapDoesNotFallbackToSystemPython(t *testing.T) {
	rootDir := t.TempDir()
	requirementsPath := filepath.Join(rootDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("qwen-asr==0.0.6\n"), 0o644); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	service := NewServiceAtRoot(
		filepath.Join(rootDir, "managed"),
		WithRequirementsPath(requirementsPath),
		WithWorkerScriptPath(filepath.Join(rootDir, "worker.py")),
	)

	_, err := service.EnsureReady(context.Background())
	if err == nil {
		t.Fatal("expected missing managed runtime source error")
	}

	if !errors.Is(err, ErrManagedRuntimeUnavailable) {
		t.Fatalf("expected managed runtime availability error, got %v", err)
	}
}

func TestRuntimeBootstrapReportsTimeoutWithTrimmedPipOutput(t *testing.T) {
	rootDir := t.TempDir()
	requirementsPath := filepath.Join(rootDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("qwen-asr==0.0.6\n"), 0o644); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	service := NewServiceAtRoot(
		filepath.Join(rootDir, "managed"),
		WithManagedRuntimeSource(writeSleepingRuntimeSource(t)),
		WithRequirementsPath(requirementsPath),
		WithWorkerScriptPath(filepath.Join(rootDir, "worker.py")),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := service.EnsureReady(ctx)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	message := err.Error()
	if !strings.Contains(message, "exceeded the 30 minute setup window") {
		t.Fatalf("expected timeout message, got %s", message)
	}
	if strings.Contains(message, "line-01") || strings.Contains(message, "line-24") {
		t.Fatalf("expected timeout message to avoid raw pip transcript, got %s", message)
	}
}

func TestRuntimeBootstrapReportsCancellationWithTrimmedPipOutput(t *testing.T) {
	rootDir := t.TempDir()
	requirementsPath := filepath.Join(rootDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("qwen-asr==0.0.6\n"), 0o644); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	service := NewServiceAtRoot(
		filepath.Join(rootDir, "managed"),
		WithManagedRuntimeSource(writeSleepingRuntimeSource(t)),
		WithRequirementsPath(requirementsPath),
		WithWorkerScriptPath(filepath.Join(rootDir, "worker.py")),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.EnsureReady(ctx)
	if err == nil {
		t.Fatal("expected cancellation error")
	}

	message := err.Error()
	if !strings.Contains(message, "installation was canceled") {
		t.Fatalf("expected cancellation message, got %s", message)
	}
	if strings.Contains(message, "line-01") || strings.Contains(message, "line-24") {
		t.Fatalf("expected cancellation message to avoid raw pip transcript, got %s", message)
	}
}

func TestSummarizeCommandOutputKeepsOnlyRecentLines(t *testing.T) {
	var output strings.Builder
	for index := 1; index <= 24; index++ {
		output.WriteString(fmt.Sprintf("line-%02d\n", index))
	}

	summary := summarizeCommandOutput([]byte(output.String()))
	if strings.Contains(summary, "line-01") {
		t.Fatalf("expected oldest line to be trimmed, got %s", summary)
	}
	if !strings.Contains(summary, "line-13") || !strings.Contains(summary, "line-24") {
		t.Fatalf("expected most recent lines to remain, got %s", summary)
	}
}

func TestRuntimeStatusUsesBundledResourcesBeforeRepoPaths(t *testing.T) {
	rootDir := t.TempDir()
	resourceRoot := t.TempDir()
	runtimeRoot := filepath.Join(resourceRoot, "runtime")
	if err := os.MkdirAll(filepath.Join(runtimeRoot, "python", "bin"), 0o755); err != nil {
		t.Fatalf("create bundled runtime: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeRoot, "python", "bin", "python3"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write bundled python: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeRoot, "worker.py"), []byte("print('worker')\n"), 0o644); err != nil {
		t.Fatalf("write bundled worker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeRoot, "requirements.txt"), []byte("qwen-asr==0.0.6\n"), 0o644); err != nil {
		t.Fatalf("write bundled requirements: %v", err)
	}

	t.Setenv("ASRSUBS_RESOURCE_ROOT", resourceRoot)

	service, err := newService(filepath.Join(rootDir, "managed"))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	status := service.Status()
	if status.State != "missing" {
		t.Fatalf("expected missing state before preparation, got %s", status.State)
	}
	if status.WorkerPath != filepath.Join(runtimeRoot, "worker.py") {
		t.Fatalf("expected bundled worker path, got %s", status.WorkerPath)
	}

	source, err := service.resolveManagedRuntimeSource()
	if err != nil {
		t.Fatalf("resolve managed runtime source: %v", err)
	}
	if source != filepath.Join(runtimeRoot, "python") {
		t.Fatalf("expected bundled runtime source, got %s", source)
	}
}

func writeSleepingRuntimeSource(t *testing.T) string {
	t.Helper()

	rootDir := t.TempDir()
	binDir := filepath.Join(rootDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create runtime bin dir: %v", err)
	}

	var pipLines strings.Builder
	for index := 1; index <= 24; index++ {
		pipLines.WriteString("echo line-")
		pipLines.WriteString(fmt.Sprintf("%02d", index))
		pipLines.WriteString(" 1>&2\n")
	}

	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-m\" ] && [ \"$2\" = \"pip\" ]; then\n" +
		pipLines.String() +
		"  sleep 5\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 0\n"

	pythonPath := filepath.Join(binDir, "python3")
	if err := os.WriteFile(pythonPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake python: %v", err)
	}

	return rootDir
}
