package transcription

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"ASRSubs/internal/models"
	asrruntime "ASRSubs/internal/runtime"
)

func TestTranscriptionRunsStagesToWorker(t *testing.T) {
	modelService := readyModelService(t)
	service := NewService(
		newFakeRuntimeService(t),
		modelService,
		WithMediaPreparer(func(_ context.Context, inputPath string, outputPath string) error {
			return os.WriteFile(outputPath, []byte(inputPath), 0o644)
		}),
		WithWorkerRunner(func(_ context.Context, request asrruntime.WorkerRequest) (asrruntime.WorkerResponse, error) {
			if request.Command != "transcribe" {
				t.Fatalf("unexpected worker command: %s", request.Command)
			}
			if request.AudioPath == "" || request.ModelPath == "" {
				t.Fatal("expected prepared audio and model path")
			}
			return asrruntime.WorkerResponse{OK: true, Command: "transcribe"}, nil
		}),
	)

	stages := make([]string, 0, 3)
	err := service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "clip.wav"),
		ModelID:   "Qwen3-ASR-1.7B",
	}, func(snapshot Snapshot) {
		stages = append(stages, snapshot.Stage)
	})
	if err != nil {
		t.Fatalf("start transcription: %v", err)
	}

	expected := []string{StagePreparingMedia, StageTranscribing}
	if len(stages) != len(expected) {
		t.Fatalf("unexpected stages: %v", stages)
	}
	for index, stage := range expected {
		if stages[index] != stage {
			t.Fatalf("expected stage %s at %d, got %s", stage, index, stages[index])
		}
	}
}

func TestTranscriptionDownloadsMissingModelBeforeWorker(t *testing.T) {
	modelService := models.NewServiceAtRoot(t.TempDir(), nil, models.WithDownloader(func(_ context.Context, model models.ModelDescriptor, destination string) error {
		return os.WriteFile(filepath.Join(destination, "weights.bin"), []byte(model.ID), 0o644)
	}))

	service := NewService(
		newFakeRuntimeService(t),
		modelService,
		WithMediaPreparer(func(_ context.Context, inputPath string, outputPath string) error {
			return os.WriteFile(outputPath, []byte(inputPath), 0o644)
		}),
		WithWorkerRunner(func(_ context.Context, request asrruntime.WorkerRequest) (asrruntime.WorkerResponse, error) {
			return asrruntime.WorkerResponse{OK: true, Command: request.Command}, nil
		}),
	)

	stages := []string{}
	if err := service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "clip.wav"),
		ModelID:   "Qwen3-ASR-0.6B",
	}, func(snapshot Snapshot) {
		stages = append(stages, snapshot.Stage)
	}); err != nil {
		t.Fatalf("start transcription: %v", err)
	}

	if len(stages) != 3 {
		t.Fatalf("expected 3 stages, got %v", stages)
	}
	if stages[1] != StageDownloading {
		t.Fatalf("expected download stage, got %v", stages)
	}
}

func TestTranscriptionReturnsFailureSummary(t *testing.T) {
	service := NewService(
		newFakeRuntimeService(t),
		readyModelService(t),
		WithMediaPreparer(func(context.Context, string, string) error {
			return errors.New("ffmpeg missing")
		}),
	)

	err := service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "clip.wav"),
		ModelID:   "Qwen3-ASR-1.7B",
	}, func(Snapshot) {})
	if err == nil {
		t.Fatal("expected failure")
	}

	failure, ok := err.(*Failure)
	if !ok {
		t.Fatalf("expected Failure, got %T", err)
	}
	if failure.Summary != "Media preparation failed." {
		t.Fatalf("unexpected summary: %s", failure.Summary)
	}
}

func readyModelService(t *testing.T) *models.Service {
	t.Helper()

	service := models.NewServiceAtRoot(t.TempDir(), nil, models.WithDownloader(func(_ context.Context, model models.ModelDescriptor, destination string) error {
		return os.WriteFile(filepath.Join(destination, "weights.bin"), []byte(model.ID), 0o644)
	}))

	if _, err := service.StartDownload("Qwen3-ASR-1.7B"); err != nil {
		t.Fatalf("start download: %v", err)
	}
	if _, err := service.EnsureReady(context.Background(), "Qwen3-ASR-1.7B"); err != nil {
		t.Fatalf("ensure model ready: %v", err)
	}

	return service
}

func writeFakeTranscriptionRuntime(t *testing.T) string {
	t.Helper()

	rootDir := t.TempDir()
	binDir := filepath.Join(rootDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake runtime dir: %v", err)
	}

	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"-m\" ] && [ \"$2\" = \"pip\" ]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"payload=$(cat)\n" +
		"case \"$payload\" in\n" +
		"  *'\"command\":\"smoke\"'*)\n" +
		"    echo '{\"ok\":true,\"command\":\"smoke\",\"message\":\"Managed runtime worker is ready.\"}'\n" +
		"    exit 0\n" +
		"    ;;\n" +
		"esac\n" +
		"echo '{\"ok\":true,\"command\":\"transcribe\",\"message\":\"Transcription contract accepted.\"}'\n" +
		"exit 0\n"

	pythonPath := filepath.Join(binDir, "python3")
	if err := os.WriteFile(pythonPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake python: %v", err)
	}

	return rootDir
}

func newFakeRuntimeService(t *testing.T) *asrruntime.Service {
	t.Helper()

	requirementsPath, err := filepath.Abs(filepath.Join("..", "runtime", "requirements.txt"))
	if err != nil {
		t.Fatalf("resolve requirements path: %v", err)
	}

	workerPath, err := filepath.Abs(filepath.Join("..", "runtime", "worker.py"))
	if err != nil {
		t.Fatalf("resolve worker path: %v", err)
	}

	return asrruntime.NewServiceAtRoot(
		t.TempDir(),
		asrruntime.WithManagedRuntimeSource(writeFakeTranscriptionRuntime(t)),
		asrruntime.WithRequirementsPath(requirementsPath),
		asrruntime.WithWorkerScriptPath(workerPath),
	)
}
