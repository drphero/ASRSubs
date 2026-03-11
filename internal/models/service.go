package models

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	asrruntime "ASRSubs/internal/runtime"
)

const (
	StateNotDownloaded = "not_downloaded"
	StateDownloading   = "downloading"
	StateReady         = "ready"
	StateFailed        = "failed"
)

type DownloadFunc func(ctx context.Context, model ModelDescriptor, destination string) error

type Option func(*serviceConfig)

type serviceConfig struct {
	rootDir  string
	download DownloadFunc
	emitter  func(Snapshot)
}

type persistedState struct {
	Failures map[string]string `json:"failures"`
}

type Snapshot struct {
	Version int           `json:"version"`
	Models  []ModelStatus `json:"models"`
}

type ModelStatus struct {
	ModelDescriptor
	State      string `json:"state"`
	StateLabel string `json:"stateLabel"`
	Path       string `json:"path"`
	Error      string `json:"error,omitempty"`
}

type Service struct {
	rootDir  string
	download DownloadFunc
	emitter  func(Snapshot)
	runtime  *asrruntime.Service

	mu       sync.RWMutex
	active   map[string]context.CancelFunc
	failures map[string]string
}

func WithRootDir(path string) Option {
	return func(cfg *serviceConfig) {
		cfg.rootDir = path
	}
}

func WithDownloader(download DownloadFunc) Option {
	return func(cfg *serviceConfig) {
		cfg.download = download
	}
}

func WithStateEmitter(emitter func(Snapshot)) Option {
	return func(cfg *serviceConfig) {
		cfg.emitter = emitter
	}
}

func NewService(appName string, runtime *asrruntime.Service, options ...Option) (*Service, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	rootDir := filepath.Join(configDir, appName, "models")
	return newService(rootDir, runtime, options...)
}

func NewServiceAtRoot(rootDir string, runtime *asrruntime.Service, options ...Option) *Service {
	service, err := newService(rootDir, runtime, options...)
	if err != nil {
		panic(err)
	}

	return service
}

func newService(rootDir string, runtime *asrruntime.Service, options ...Option) (*Service, error) {
	cfg := serviceConfig{
		rootDir: rootDir,
	}
	for _, option := range options {
		option(&cfg)
	}

	service := &Service{
		rootDir: rootDir,
		emitter: cfg.emitter,
		runtime: runtime,
		active:  map[string]context.CancelFunc{},
	}

	if cfg.download != nil {
		service.download = cfg.download
	} else {
		service.download = service.defaultDownload
	}

	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, err
	}

	state, err := service.loadState()
	if err != nil {
		return nil, err
	}
	service.failures = state.Failures
	if service.failures == nil {
		service.failures = map[string]string{}
	}

	return service, nil
}

func (s *Service) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotLocked()
}

func (s *Service) GetModelState(id string) (ModelStatus, error) {
	model, ok := Lookup(id)
	if !ok {
		return ModelStatus{}, fmt.Errorf("unknown model: %s", id)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLocked(model), nil
}

func (s *Service) StartDownload(id string) (ModelStatus, error) {
	model, ok := Lookup(id)
	if !ok {
		return ModelStatus{}, fmt.Errorf("unknown model: %s", id)
	}

	s.mu.Lock()
	if _, active := s.active[id]; active {
		status := s.statusLocked(model)
		s.mu.Unlock()
		return status, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.active[id] = cancel
	delete(s.failures, id)
	status := s.statusLocked(model)
	snapshot := s.snapshotLocked()
	s.mu.Unlock()

	s.emit(snapshot)

	go s.runDownload(ctx, model)
	return status, nil
}

func (s *Service) Delete(id string) (ModelStatus, error) {
	model, ok := Lookup(id)
	if !ok {
		return ModelStatus{}, fmt.Errorf("unknown model: %s", id)
	}

	s.mu.Lock()
	if _, active := s.active[id]; active {
		s.mu.Unlock()
		return ModelStatus{}, fmt.Errorf("model is still downloading: %s", id)
	}

	if err := os.RemoveAll(s.modelDir(id)); err != nil {
		s.mu.Unlock()
		return ModelStatus{}, fmt.Errorf("model files could not be deleted: %w", err)
	}

	delete(s.failures, id)
	if err := s.saveStateLocked(); err != nil {
		s.mu.Unlock()
		return ModelStatus{}, err
	}

	status := s.statusLocked(model)
	snapshot := s.snapshotLocked()
	s.mu.Unlock()

	s.emit(snapshot)
	return status, nil
}

func (s *Service) EnsureReady(ctx context.Context, id string) (ModelStatus, error) {
	status, err := s.GetModelState(id)
	if err != nil {
		return ModelStatus{}, err
	}

	if status.State == StateReady {
		return status, nil
	}

	if status.State != StateDownloading {
		if _, err := s.StartDownload(id); err != nil {
			return ModelStatus{}, err
		}
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ModelStatus{}, ctx.Err()
		case <-ticker.C:
			next, err := s.GetModelState(id)
			if err != nil {
				return ModelStatus{}, err
			}
			if next.State == StateReady {
				return next, nil
			}
			if next.State == StateFailed {
				message := next.Error
				if message == "" {
					message = "model download failed"
				}
				return ModelStatus{}, fmt.Errorf(message)
			}
		}
	}
}

func (s *Service) runDownload(ctx context.Context, model ModelDescriptor) {
	destination := s.modelDir(model.ID)
	_ = os.RemoveAll(destination)
	_ = os.MkdirAll(destination, 0o755)

	err := s.download(ctx, model, destination)

	s.mu.Lock()
	delete(s.active, model.ID)

	if err != nil {
		_ = os.RemoveAll(destination)
		s.failures[model.ID] = err.Error()
	} else {
		delete(s.failures, model.ID)
		_ = s.writeReadyMarker(model)
	}

	_ = s.saveStateLocked()
	snapshot := s.snapshotLocked()
	s.mu.Unlock()

	s.emit(snapshot)
}

func (s *Service) snapshotLocked() Snapshot {
	models := Catalog()
	statuses := make([]ModelStatus, 0, len(models))
	for _, model := range models {
		statuses = append(statuses, s.statusLocked(model))
	}

	return Snapshot{
		Version: 1,
		Models:  statuses,
	}
}

func (s *Service) statusLocked(model ModelDescriptor) ModelStatus {
	status := ModelStatus{
		ModelDescriptor: model,
		State:           StateNotDownloaded,
		StateLabel:      stateLabel(StateNotDownloaded),
		Path:            s.modelDir(model.ID),
		Error:           s.failures[model.ID],
	}

	if _, active := s.active[model.ID]; active {
		status.State = StateDownloading
		status.StateLabel = stateLabel(StateDownloading)
		status.Error = ""
		return status
	}

	if s.readyMarkerExists(model.ID) {
		status.State = StateReady
		status.StateLabel = stateLabel(StateReady)
		status.Error = ""
		return status
	}

	if status.Error != "" {
		status.State = StateFailed
		status.StateLabel = stateLabel(StateFailed)
	}

	return status
}

func (s *Service) defaultDownload(ctx context.Context, model ModelDescriptor, destination string) error {
	if s.runtime == nil {
		return fmt.Errorf("managed runtime is not ready")
	}

	if _, err := s.runtime.EnsureReady(ctx); err != nil {
		return err
	}

	script := strings.Join([]string{
		"import sys",
		"from huggingface_hub import snapshot_download",
		"snapshot_download(repo_id=sys.argv[1], local_dir=sys.argv[2])",
	}, "; ")

	cmd := exec.CommandContext(ctx, s.runtime.PythonPath(), "-c", script, model.RepoID, destination)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("model download failed for %s: %s", model.ID, message)
	}

	return nil
}

func (s *Service) emit(snapshot Snapshot) {
	if s.emitter != nil {
		s.emitter(snapshot)
	}
}

func (s *Service) loadState() (persistedState, error) {
	data, err := os.ReadFile(s.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return persistedState{Failures: map[string]string{}}, nil
		}
		return persistedState{}, err
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return persistedState{}, err
	}

	if state.Failures == nil {
		state.Failures = map[string]string{}
	}

	return state, nil
}

func (s *Service) saveStateLocked() error {
	state := persistedState{
		Failures: s.failures,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.statePath(), data, 0o644)
}

func (s *Service) writeReadyMarker(model ModelDescriptor) error {
	payload := map[string]string{
		"id":        model.ID,
		"repoId":    model.RepoID,
		"completed": time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.readyMarkerPath(model.ID), data, 0o644)
}

func (s *Service) readyMarkerExists(id string) bool {
	info, err := os.Stat(s.readyMarkerPath(id))
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func (s *Service) statePath() string {
	return filepath.Join(s.rootDir, "state.json")
}

func (s *Service) modelDir(id string) string {
	return filepath.Join(s.rootDir, id)
}

func (s *Service) readyMarkerPath(id string) string {
	return filepath.Join(s.modelDir(id), ".asrsubs-ready.json")
}

func stateLabel(state string) string {
	switch state {
	case StateDownloading:
		return "Downloading"
	case StateReady:
		return "Ready"
	case StateFailed:
		return "Failed"
	default:
		return "Not downloaded"
	}
}
