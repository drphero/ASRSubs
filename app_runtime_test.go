package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	asrruntime "ASRSubs/internal/runtime"
)

func TestEnsureRuntimeReadyRunsManagedSmokePath(t *testing.T) {
	rootDir := t.TempDir()
	requirementsPath := filepath.Join(rootDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("qwen-asr==0.0.6\n"), 0o644); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	t.Setenv("ASRSUBS_FAKE_PIP_LOG", filepath.Join(rootDir, "pip.log"))

	app := &App{}
	app.initDiagnostics()
	app.runtime = asrruntime.NewServiceAtRoot(
		filepath.Join(rootDir, "managed"),
		asrruntime.WithManagedRuntimeSource(writeFakeManagedRuntimeSource(t)),
		asrruntime.WithRequirementsPath(requirementsPath),
		asrruntime.WithWorkerScriptPath(filepath.Join(rootDir, "worker.py")),
	)

	readiness, err := app.EnsureRuntimeReady()
	if err != nil {
		t.Fatalf("ensure runtime ready: %v", err)
	}

	if readiness.State != "ready" {
		t.Fatalf("expected ready state, got %s", readiness.State)
	}

	snapshot := app.GetDiagnosticsSnapshot()
	if len(snapshot.Entries) == 0 {
		t.Fatal("expected diagnostics entry")
	}

	last := snapshot.Entries[len(snapshot.Entries)-1]
	if last.Message != "Managed runtime is ready." {
		t.Fatalf("unexpected diagnostic message: %s", last.Message)
	}
}

func TestGetRuntimeReadinessReportsMissingBeforePreparation(t *testing.T) {
	rootDir := t.TempDir()
	requirementsPath := filepath.Join(rootDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("qwen-asr==0.0.6\n"), 0o644); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	app := &App{}
	app.runtime = asrruntime.NewServiceAtRoot(
		filepath.Join(rootDir, "managed"),
		asrruntime.WithManagedRuntimeSource(writeFakeManagedRuntimeSource(t)),
		asrruntime.WithRequirementsPath(requirementsPath),
		asrruntime.WithWorkerScriptPath(filepath.Join(rootDir, "worker.py")),
	)

	if err := os.WriteFile(filepath.Join(rootDir, "worker.py"), []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("write worker: %v", err)
	}

	readiness, err := app.GetRuntimeReadiness()
	if err != nil {
		t.Fatalf("get runtime readiness: %v", err)
	}

	if readiness.State != "missing" {
		t.Fatalf("expected missing state, got %s", readiness.State)
	}
}

func TestEnsureRuntimeReadyReportsConciseTimeoutFailure(t *testing.T) {
	rootDir := t.TempDir()
	requirementsPath := filepath.Join(rootDir, "requirements.txt")
	if err := os.WriteFile(requirementsPath, []byte("qwen-asr==0.0.6\n"), 0o644); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	app := &App{}
	app.initDiagnostics()
	app.runtime = asrruntime.NewServiceAtRoot(
		filepath.Join(rootDir, "managed"),
		asrruntime.WithManagedRuntimeSource(writeSlowManagedRuntimeSource(t)),
		asrruntime.WithRequirementsPath(requirementsPath),
		asrruntime.WithWorkerScriptPath(filepath.Join(rootDir, "worker.py")),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	status, err := app.ensureRuntimeReadyWithContext(ctx)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	if status.State != "failed" {
		t.Fatalf("expected failed state, got %s", status.State)
	}
	if !strings.Contains(status.Detail, "exceeded the 30 minute setup window") {
		t.Fatalf("expected concise timeout detail, got %s", status.Detail)
	}

	snapshot := app.GetDiagnosticsSnapshot()
	if len(snapshot.Entries) == 0 {
		t.Fatal("expected diagnostics entry")
	}
}

func writeFakeManagedRuntimeSource(t *testing.T) string {
	t.Helper()

	rootDir := t.TempDir()
	binDir := filepath.Join(rootDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake runtime dir: %v", err)
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
		"  *'\"command\":\"smoke\"'*)\n" +
		"    echo '{\"ok\":true,\"command\":\"smoke\",\"message\":\"Managed runtime worker is ready.\"}'\n" +
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

func writeSlowManagedRuntimeSource(t *testing.T) string {
	t.Helper()

	rootDir := t.TempDir()
	binDir := filepath.Join(rootDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake runtime dir: %v", err)
	}

	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-m\" ] && [ \"$2\" = \"pip\" ]; then\n" +
		"  echo 'Downloading packages' 1>&2\n" +
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
