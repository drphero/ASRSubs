package transcription

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ASRSubs/internal/models"
	asrruntime "ASRSubs/internal/runtime"
)

const (
	StagePreparingMedia = "Preparing media"
	StageDownloading    = "Downloading model"
	StageTranscribing   = "Transcribing"
)

type MediaPreparer func(ctx context.Context, inputPath string, outputPath string) error
type WorkerRunner func(ctx context.Context, request asrruntime.WorkerRequest) (asrruntime.WorkerResponse, error)
type Option func(*Service)

type StartRequest struct {
	MediaPath string `json:"mediaPath"`
	ModelID   string `json:"modelID"`
}

type Snapshot struct {
	Active         bool   `json:"active"`
	CanRetry       bool   `json:"canRetry"`
	Stage          string `json:"stage"`
	FilePath       string `json:"filePath"`
	FileName       string `json:"fileName"`
	ModelID        string `json:"modelID"`
	FailureSummary string `json:"failureSummary"`
}

type Failure struct {
	Summary string
	Detail  string
	Cause   error
}

type Service struct {
	runtime      *asrruntime.Service
	models       *models.Service
	prepareMedia MediaPreparer
	runWorker    WorkerRunner
	tempDir      string
}

func (f *Failure) Error() string {
	if f.Detail != "" {
		return f.Detail
	}
	return f.Summary
}

func (f *Failure) Unwrap() error {
	return f.Cause
}

func WithMediaPreparer(preparer MediaPreparer) Option {
	return func(service *Service) {
		service.prepareMedia = preparer
	}
}

func WithWorkerRunner(runner WorkerRunner) Option {
	return func(service *Service) {
		service.runWorker = runner
	}
}

func WithTempDir(path string) Option {
	return func(service *Service) {
		service.tempDir = path
	}
}

func NewService(runtime *asrruntime.Service, modelService *models.Service, options ...Option) *Service {
	service := &Service{
		runtime: runtime,
		models:  modelService,
		tempDir: os.TempDir(),
	}

	service.prepareMedia = service.defaultPrepareMedia
	service.runWorker = service.defaultRunWorker

	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) Start(ctx context.Context, request StartRequest, emit func(Snapshot)) error {
	if strings.TrimSpace(request.MediaPath) == "" {
		return &Failure{Summary: "Choose a media file before starting.", Detail: "transcription request did not include a media file"}
	}
	if strings.TrimSpace(request.ModelID) == "" {
		return &Failure{Summary: "Choose a model before starting.", Detail: "transcription request did not include a model"}
	}

	base := Snapshot{
		Active:   true,
		CanRetry: false,
		FilePath: request.MediaPath,
		FileName: filepath.Base(request.MediaPath),
		ModelID:  request.ModelID,
	}

	emit(withStage(base, StagePreparingMedia))

	if s.runtime == nil {
		return &Failure{Summary: "Managed runtime could not be prepared.", Detail: "runtime service is not configured"}
	}
	if _, err := s.runtime.EnsureReady(ctx); err != nil {
		return &Failure{Summary: "Managed runtime could not be prepared.", Detail: err.Error(), Cause: err}
	}

	preparedPath, cleanup, err := s.prepare(ctx, request.MediaPath)
	if err != nil {
		return &Failure{Summary: "Media preparation failed.", Detail: err.Error(), Cause: err}
	}
	defer cleanup()

	modelStatus, err := s.models.GetModelState(request.ModelID)
	if err != nil {
		return &Failure{Summary: "Model state could not be loaded.", Detail: err.Error(), Cause: err}
	}

	if modelStatus.State != models.StateReady {
		emit(withStage(base, StageDownloading))
		modelStatus, err = s.models.EnsureReady(ctx, request.ModelID)
		if err != nil {
			return &Failure{Summary: "Model download failed.", Detail: err.Error(), Cause: err}
		}
	}

	emit(withStage(base, StageTranscribing))
	if _, err := s.runWorker(ctx, asrruntime.WorkerRequest{
		Command:   "transcribe",
		AudioPath: preparedPath,
		ModelPath: modelStatus.Path,
	}); err != nil {
		return &Failure{Summary: "Local transcription failed.", Detail: err.Error(), Cause: err}
	}

	return nil
}

func withStage(snapshot Snapshot, stage string) Snapshot {
	snapshot.Stage = stage
	return snapshot
}

func (s *Service) prepare(ctx context.Context, inputPath string) (string, func(), error) {
	tempDir, err := os.MkdirTemp(s.tempDir, "asrsubs-transcription-*")
	if err != nil {
		return "", func() {}, err
	}

	outputPath := filepath.Join(tempDir, "prepared.wav")
	if err := s.prepareMedia(ctx, inputPath, outputPath); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", func() {}, err
	}

	return outputPath, func() {
		_ = os.RemoveAll(tempDir)
	}, nil
}

func (s *Service) defaultPrepareMedia(ctx context.Context, inputPath string, outputPath string) error {
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i",
		inputPath,
		"-vn",
		"-ac",
		"1",
		"-ar",
		"16000",
		"-c:a",
		"pcm_s16le",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("ffmpeg could not prepare the selected media: %s", message)
	}

	return nil
}

func (s *Service) defaultRunWorker(ctx context.Context, request asrruntime.WorkerRequest) (asrruntime.WorkerResponse, error) {
	return s.runtime.RunWorker(ctx, request)
}
