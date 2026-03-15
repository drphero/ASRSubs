package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
