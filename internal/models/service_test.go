package models

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCatalogIncludesSupportedModels(t *testing.T) {
	models := Catalog()
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	large, ok := Lookup("Qwen3-ASR-1.7B")
	if !ok {
		t.Fatal("expected large model in catalog")
	}

	if !large.Default {
		t.Fatal("expected Qwen3-ASR-1.7B to be the default model")
	}
}

func TestModelDownloadTransitionsToReady(t *testing.T) {
	service := NewServiceAtRoot(t.TempDir(), nil, WithDownloader(func(_ context.Context, model ModelDescriptor, destination string) error {
		return os.WriteFile(filepath.Join(destination, "weights.bin"), []byte(model.ID), 0o644)
	}))

	status, err := service.StartDownload("Qwen3-ASR-0.6B")
	if err != nil {
		t.Fatalf("start download: %v", err)
	}

	if status.State != StateDownloading {
		t.Fatalf("expected downloading state, got %s", status.State)
	}

	waitForState(t, service, "Qwen3-ASR-0.6B", StateReady)
}

func TestModelDownloadTransitionsToFailed(t *testing.T) {
	service := NewServiceAtRoot(t.TempDir(), nil, WithDownloader(func(context.Context, ModelDescriptor, string) error {
		return errors.New("network unavailable")
	}))

	if _, err := service.StartDownload("Qwen3-ASR-1.7B"); err != nil {
		t.Fatalf("start download: %v", err)
	}

	status := waitForState(t, service, "Qwen3-ASR-1.7B", StateFailed)
	if status.Error != "network unavailable" {
		t.Fatalf("unexpected failure message: %s", status.Error)
	}
}

func TestDeleteRemovesReadyModel(t *testing.T) {
	service := NewServiceAtRoot(t.TempDir(), nil, WithDownloader(func(_ context.Context, model ModelDescriptor, destination string) error {
		return os.WriteFile(filepath.Join(destination, "weights.bin"), []byte(model.ID), 0o644)
	}))

	if _, err := service.StartDownload("Qwen3-ASR-1.7B"); err != nil {
		t.Fatalf("start download: %v", err)
	}

	waitForState(t, service, "Qwen3-ASR-1.7B", StateReady)

	status, err := service.Delete("Qwen3-ASR-1.7B")
	if err != nil {
		t.Fatalf("delete model: %v", err)
	}

	if status.State != StateNotDownloaded {
		t.Fatalf("expected not downloaded after delete, got %s", status.State)
	}
}

func waitForState(t *testing.T, service *Service, modelID string, expected string) ModelStatus {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, err := service.GetModelState(modelID)
		if err != nil {
			t.Fatalf("get state: %v", err)
		}

		if status.State == expected {
			return status
		}

		time.Sleep(10 * time.Millisecond)
	}

	status, err := service.GetModelState(modelID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}

	t.Fatalf("timed out waiting for %s, got %s", expected, status.State)
	return ModelStatus{}
}
