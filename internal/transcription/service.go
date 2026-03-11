package transcription

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

	"ASRSubs/internal/models"
	asrruntime "ASRSubs/internal/runtime"
)

const (
	StagePreparingMedia    = "Preparing media"
	StageDownloading       = "Downloading model"
	StageTranscribing      = "Transcribing"
	StageAligning          = "Aligning"
	StageBuildingSubtitles = "Building subtitles"
)

type MediaPreparer func(ctx context.Context, inputPath string, outputPath string) error
type MediaDurationProber func(ctx context.Context, inputPath string) (time.Duration, error)
type MediaSegmenter func(ctx context.Context, inputPath string, outputPath string, start time.Duration, duration time.Duration) error
type WorkerRunner func(ctx context.Context, request asrruntime.WorkerRequest) (asrruntime.WorkerResponse, error)
type SubtitleBuilder func(words []WordTimestamp, prefs RunPreferences) ([]SubtitleSegment, error)
type Option func(*Service)

type RunPreferences struct {
	MaxLineLength         int  `json:"maxLineLength"`
	LinesPerSubtitle      int  `json:"linesPerSubtitle"`
	AlignmentChunkMinutes int  `json:"alignmentChunkMinutes"`
	OneWordPerSubtitle    bool `json:"oneWordPerSubtitle"`
}

type StartRequest struct {
	MediaPath   string         `json:"mediaPath"`
	ModelID     string         `json:"modelID"`
	Preferences RunPreferences `json:"preferences"`
}

type Snapshot struct {
	Active         bool   `json:"active"`
	CanRetry       bool   `json:"canRetry"`
	Stage          string `json:"stage"`
	FailedStage    string `json:"failedStage"`
	PartIndex      int    `json:"partIndex"`
	PartCount      int    `json:"partCount"`
	FilePath       string `json:"filePath"`
	FileName       string `json:"fileName"`
	ModelID        string `json:"modelID"`
	FailureSummary string `json:"failureSummary"`
}

type Failure struct {
	Stage     string
	Summary   string
	Detail    string
	Retryable bool
	PartIndex int
	PartCount int
	Cause     error
}

type runState struct {
	Fingerprint    string       `json:"fingerprint"`
	Request        StartRequest `json:"request"`
	WorkDir        string       `json:"workDir"`
	PreparedPath   string       `json:"preparedPath"`
	DurationMS     int          `json:"durationMs"`
	ChunkPlan      []ChunkPlan  `json:"chunkPlan,omitempty"`
	FailedStage    string       `json:"failedStage,omitempty"`
	FailedChunk    int          `json:"failedChunk,omitempty"`
	TranscriptPath string       `json:"transcriptPath,omitempty"`
	AlignmentPath  string       `json:"alignmentPath,omitempty"`
	TimelinePath   string       `json:"timelinePath,omitempty"`
}

type transcriptArtifact struct {
	Text     string                       `json:"text"`
	Language string                       `json:"language,omitempty"`
	Words    []asrruntime.TranscriptToken `json:"words"`
}

type Service struct {
	runtime        *asrruntime.Service
	models         *models.Service
	prepareMedia   MediaPreparer
	probeDuration  MediaDurationProber
	segmentMedia   MediaSegmenter
	runWorker      WorkerRunner
	buildSubtitles SubtitleBuilder
	tempDir        string

	mu      sync.Mutex
	lastRun *runState
}

func (s *Service) GetLatestSubtitleDraft() (SubtitleDraft, error) {
	s.mu.Lock()
	run := s.lastRun
	s.mu.Unlock()

	if run == nil {
		return SubtitleDraft{}, fmt.Errorf("no transcription draft is available yet")
	}
	if !fileExists(run.TimelinePath) {
		return SubtitleDraft{}, fmt.Errorf("subtitle timeline is not available yet")
	}

	var timeline Timeline
	if err := readJSON(run.TimelinePath, &timeline); err != nil {
		return SubtitleDraft{}, err
	}

	return SubtitleDraft{
		Text:              SerializeSRT(timeline.Subtitles),
		SuggestedFilename: DraftFilenameForMedia(run.Request.MediaPath),
		SourceFilePath:    run.Request.MediaPath,
		SourceFileName:    filepath.Base(run.Request.MediaPath),
	}, nil
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

func WithDurationProber(prober MediaDurationProber) Option {
	return func(service *Service) {
		service.probeDuration = prober
	}
}

func WithMediaSegmenter(segmenter MediaSegmenter) Option {
	return func(service *Service) {
		service.segmentMedia = segmenter
	}
}

func WithWorkerRunner(runner WorkerRunner) Option {
	return func(service *Service) {
		service.runWorker = runner
	}
}

func WithSubtitleBuilder(builder SubtitleBuilder) Option {
	return func(service *Service) {
		service.buildSubtitles = builder
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
	service.probeDuration = service.defaultProbeDuration
	service.segmentMedia = service.defaultSegmentMedia
	service.runWorker = service.defaultRunWorker
	service.buildSubtitles = BuildSubtitles

	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) Start(ctx context.Context, request StartRequest, emit func(Snapshot)) error {
	if strings.TrimSpace(request.MediaPath) == "" {
		return &Failure{Stage: StagePreparingMedia, Summary: "Choose a media file before starting.", Detail: "transcription request did not include a media file", Retryable: true}
	}
	if strings.TrimSpace(request.ModelID) == "" {
		return &Failure{Stage: StagePreparingMedia, Summary: "Choose a model before starting.", Detail: "transcription request did not include a model", Retryable: true}
	}

	request.Preferences = sanitizePreferences(request.Preferences)
	base := Snapshot{
		Active:   true,
		CanRetry: true,
		FilePath: request.MediaPath,
		FileName: filepath.Base(request.MediaPath),
		ModelID:  request.ModelID,
	}

	run, err := s.prepareRun(request)
	if err != nil {
		return &Failure{Stage: StagePreparingMedia, Summary: "Working files could not be prepared.", Detail: err.Error(), Retryable: true, Cause: err}
	}

	emit(stageSnapshot(base, StagePreparingMedia, 0, len(run.ChunkPlan)))

	if s.runtime == nil {
		return s.fail(run, StagePreparingMedia, 0, len(run.ChunkPlan), "Managed runtime could not be prepared.", fmt.Errorf("runtime service is not configured"))
	}
	if s.models == nil {
		return s.fail(run, StagePreparingMedia, 0, len(run.ChunkPlan), "Model state could not be loaded.", fmt.Errorf("model service is not configured"))
	}
	if _, err := s.runtime.EnsureReady(ctx); err != nil {
		return s.fail(run, StagePreparingMedia, 0, len(run.ChunkPlan), "Managed runtime could not be prepared.", err)
	}

	if run.PreparedPath == "" {
		preparedPath := filepath.Join(run.WorkDir, "prepared.wav")
		if err := os.MkdirAll(filepath.Dir(preparedPath), 0o755); err != nil {
			return s.fail(run, StagePreparingMedia, 0, len(run.ChunkPlan), "Working files could not be prepared.", err)
		}
		if err := s.prepareMedia(ctx, request.MediaPath, preparedPath); err != nil {
			return s.fail(run, StagePreparingMedia, 0, len(run.ChunkPlan), "Media preparation failed.", err)
		}
		run.PreparedPath = preparedPath
	}

	if run.DurationMS == 0 {
		duration, err := s.probeDuration(ctx, run.PreparedPath)
		if err == nil && duration > 0 {
			run.DurationMS = int(duration / time.Millisecond)
		}
	}
	if len(run.ChunkPlan) == 0 && ShouldChunk(time.Duration(run.DurationMS)*time.Millisecond, request.Preferences) {
		run.ChunkPlan = BuildChunkPlan(run.WorkDir, time.Duration(run.DurationMS)*time.Millisecond, request.Preferences)
	}

	modelStatus, alignerStatus, needsDownload, err := s.ensureModels(ctx, request.ModelID)
	if needsDownload {
		emit(stageSnapshot(base, StageDownloading, 0, len(run.ChunkPlan)))
	}
	if err != nil {
		return s.fail(run, StageDownloading, 0, len(run.ChunkPlan), "Model download failed.", err)
	}

	if len(run.ChunkPlan) == 0 {
		return s.executeShortRun(ctx, run, base, emit, modelStatus.Path, alignerStatus.Path)
	}

	return s.executeChunkedRun(ctx, run, base, emit, modelStatus.Path, alignerStatus.Path)
}

func (s *Service) prepareRun(request StartRequest) (*runState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fingerprint := fingerprint(request)
	if s.lastRun != nil && s.lastRun.Fingerprint == fingerprint {
		return s.lastRun, nil
	}
	if s.lastRun != nil {
		_ = os.RemoveAll(s.lastRun.WorkDir)
	}

	workDir, err := os.MkdirTemp(s.tempDir, "asrsubs-run-*")
	if err != nil {
		return nil, err
	}

	run := &runState{
		Fingerprint:    fingerprint,
		Request:        request,
		WorkDir:        workDir,
		TranscriptPath: filepath.Join(workDir, "artifacts", "transcript.json"),
		AlignmentPath:  filepath.Join(workDir, "artifacts", "alignment.json"),
		TimelinePath:   filepath.Join(workDir, "artifacts", "timeline.json"),
	}
	s.lastRun = run
	return run, nil
}

func (s *Service) ensureModels(ctx context.Context, modelID string) (models.ModelStatus, models.ModelStatus, bool, error) {
	modelStatus, err := s.models.GetModelState(modelID)
	if err != nil {
		return models.ModelStatus{}, models.ModelStatus{}, false, err
	}
	alignerStatus, err := s.models.GetModelState(models.ForcedAlignerID)
	if err != nil {
		return models.ModelStatus{}, models.ModelStatus{}, false, err
	}

	needsDownload := modelStatus.State != models.StateReady || alignerStatus.State != models.StateReady

	modelStatus, err = s.models.EnsureReady(ctx, modelID)
	if err != nil {
		return models.ModelStatus{}, models.ModelStatus{}, needsDownload, err
	}
	alignerStatus, err = s.models.EnsureReady(ctx, models.ForcedAlignerID)
	if err != nil {
		return models.ModelStatus{}, models.ModelStatus{}, needsDownload, err
	}

	return modelStatus, alignerStatus, needsDownload, nil
}

func (s *Service) executeShortRun(
	ctx context.Context,
	run *runState,
	base Snapshot,
	emit func(Snapshot),
	modelPath string,
	alignerPath string,
) error {
	transcript, err := s.ensureTranscript(ctx, run, base, emit, modelPath, 0, 0)
	if err != nil {
		return err
	}

	words, err := s.ensureAlignment(ctx, run, base, emit, alignerPath, modelPath, transcript, 0, 0)
	if err != nil {
		return err
	}

	return s.ensureTimeline(run, base, emit, words, 0, 0)
}

func (s *Service) executeChunkedRun(
	ctx context.Context,
	run *runState,
	base Snapshot,
	emit func(Snapshot),
	modelPath string,
	alignerPath string,
) error {
	chunkWords := make([][]WordTimestamp, 0, len(run.ChunkPlan))
	for _, chunk := range run.ChunkPlan {
		if chunk.Index < run.FailedChunk {
			words, err := loadWords(chunk.AlignmentPath)
			if err != nil {
				return s.fail(run, StageAligning, chunk.Index, len(run.ChunkPlan), "Saved alignment state could not be loaded.", err)
			}
			chunkWords = append(chunkWords, offsetWords(words, chunk.StartMS))
			continue
		}

		if err := os.MkdirAll(filepath.Dir(chunk.AudioPath), 0o755); err != nil {
			return s.fail(run, StagePreparingMedia, chunk.Index, len(run.ChunkPlan), "Chunk working files could not be prepared.", err)
		}
		if _, err := os.Stat(chunk.AudioPath); err != nil {
			if err := s.segmentMedia(
				ctx,
				run.PreparedPath,
				chunk.AudioPath,
				time.Duration(chunk.StartMS)*time.Millisecond,
				time.Duration(chunk.DurationMS)*time.Millisecond,
			); err != nil {
				return s.fail(run, StagePreparingMedia, chunk.Index, len(run.ChunkPlan), "Media chunking failed.", err)
			}
		}

		transcript, err := loadTranscript(chunk.TranscriptPath)
		if err != nil || run.FailedStage == StageTranscribing || run.FailedChunk == chunk.Index {
			emit(stageSnapshot(base, StageTranscribing, chunk.Index, len(run.ChunkPlan)))
			for attempt := 0; attempt < 2; attempt++ {
				transcript, err = s.transcribeAudio(ctx, chunk.AudioPath, chunk.TranscriptPath, modelPath)
				if err == nil {
					break
				}
			}
			if err != nil {
				return s.fail(run, StageTranscribing, chunk.Index, len(run.ChunkPlan), "Local transcription failed.", err)
			}
		}

		words, err := loadWords(chunk.AlignmentPath)
		if err != nil || run.FailedStage == StageAligning || run.FailedChunk == chunk.Index {
			emit(stageSnapshot(base, StageAligning, chunk.Index, len(run.ChunkPlan)))
			for attempt := 0; attempt < 2; attempt++ {
				words, err = s.alignAudio(ctx, chunk.AudioPath, chunk.AlignmentPath, alignerPath, modelPath, transcript)
				if err == nil {
					break
				}
			}
			if err != nil {
				return s.fail(run, StageAligning, chunk.Index, len(run.ChunkPlan), "Timestamp alignment failed.", err)
			}
		}

		chunkWords = append(chunkWords, offsetWords(words, chunk.StartMS))
	}

	return s.ensureTimeline(run, base, emit, MergeTimeline(chunkWords), 0, len(run.ChunkPlan))
}

func (s *Service) ensureTranscript(
	ctx context.Context,
	run *runState,
	base Snapshot,
	emit func(Snapshot),
	modelPath string,
	partIndex int,
	partCount int,
) (transcriptArtifact, error) {
	if fileExists(run.TranscriptPath) && run.FailedStage != StageTranscribing {
		return loadTranscript(run.TranscriptPath)
	}

	emit(stageSnapshot(base, StageTranscribing, partIndex, partCount))
	transcript, err := s.transcribeAudio(ctx, run.PreparedPath, run.TranscriptPath, modelPath)
	if err != nil {
		return transcriptArtifact{}, s.fail(run, StageTranscribing, partIndex, partCount, "Local transcription failed.", err)
	}

	run.FailedStage = ""
	run.FailedChunk = 0
	return transcript, nil
}

func (s *Service) ensureAlignment(
	ctx context.Context,
	run *runState,
	base Snapshot,
	emit func(Snapshot),
	alignerPath string,
	modelPath string,
	transcript transcriptArtifact,
	partIndex int,
	partCount int,
) ([]WordTimestamp, error) {
	if fileExists(run.AlignmentPath) && run.FailedStage != StageAligning {
		return loadWords(run.AlignmentPath)
	}

	emit(stageSnapshot(base, StageAligning, partIndex, partCount))
	words, err := s.alignAudio(ctx, run.PreparedPath, run.AlignmentPath, alignerPath, modelPath, transcript)
	if err != nil {
		return nil, s.fail(run, StageAligning, partIndex, partCount, "Timestamp alignment failed.", err)
	}

	run.FailedStage = ""
	run.FailedChunk = 0
	return words, nil
}

func (s *Service) ensureTimeline(run *runState, base Snapshot, emit func(Snapshot), words []WordTimestamp, partIndex int, partCount int) error {
	if fileExists(run.TimelinePath) && run.FailedStage == "" {
		return nil
	}

	emit(stageSnapshot(base, StageBuildingSubtitles, partIndex, partCount))
	subtitles, err := s.buildSubtitles(words, run.Request.Preferences)
	if err != nil {
		return s.fail(run, StageBuildingSubtitles, partIndex, partCount, "Subtitle generation failed.", err)
	}

	timeline := Timeline{
		Words:     words,
		Subtitles: subtitles,
	}
	if err := writeJSON(run.TimelinePath, timeline); err != nil {
		return s.fail(run, StageBuildingSubtitles, partIndex, partCount, "Subtitle generation failed.", err)
	}

	run.FailedStage = ""
	run.FailedChunk = 0
	return nil
}

func (s *Service) transcribeAudio(ctx context.Context, audioPath string, artifactPath string, modelPath string) (transcriptArtifact, error) {
	response, err := s.runWorker(ctx, asrruntime.WorkerRequest{
		Command:   "transcribe",
		AudioPath: audioPath,
		ModelPath: modelPath,
	})
	if err != nil {
		return transcriptArtifact{}, err
	}

	var payload asrruntime.TranscriptPayload
	if err := response.DecodeDetails(&payload); err != nil {
		return transcriptArtifact{}, err
	}

	artifact := transcriptArtifact{
		Text:     payload.Text,
		Language: payload.Language,
		Words:    payload.Words,
	}
	if err := writeJSON(artifactPath, artifact); err != nil {
		return transcriptArtifact{}, err
	}

	return artifact, nil
}

func (s *Service) alignAudio(
	ctx context.Context,
	audioPath string,
	artifactPath string,
	alignerPath string,
	modelPath string,
	transcript transcriptArtifact,
) ([]WordTimestamp, error) {
	response, err := s.runWorker(ctx, asrruntime.WorkerRequest{
		Command:     "align",
		AudioPath:   audioPath,
		ModelPath:   modelPath,
		AlignerPath: alignerPath,
		Language:    transcript.Language,
		Transcript: &asrruntime.TranscriptPayload{
			Text:     transcript.Text,
			Language: transcript.Language,
			Words:    transcript.Words,
		},
	})
	if err != nil {
		return nil, err
	}

	var payload asrruntime.AlignmentPayload
	if err := response.DecodeDetails(&payload); err != nil {
		return nil, err
	}

	words := make([]WordTimestamp, 0, len(payload.Words))
	for _, word := range payload.Words {
		words = append(words, WordTimestamp{
			Text:       word.Text,
			StartMS:    word.StartMS,
			EndMS:      word.EndMS,
			Confidence: word.Confidence,
		})
	}
	if err := writeJSON(artifactPath, words); err != nil {
		return nil, err
	}

	return words, nil
}

func stageSnapshot(base Snapshot, stage string, partIndex int, partCount int) Snapshot {
	base.Stage = stage
	base.PartIndex = partIndex
	base.PartCount = partCount
	return base
}

func (s *Service) fail(run *runState, stage string, partIndex int, partCount int, summary string, err error) error {
	run.FailedStage = stage
	run.FailedChunk = partIndex
	return &Failure{
		Stage:     stage,
		Summary:   summary,
		Detail:    err.Error(),
		Retryable: true,
		PartIndex: partIndex,
		PartCount: partCount,
		Cause:     err,
	}
}

func sanitizePreferences(prefs RunPreferences) RunPreferences {
	defaults := defaultPreferences()
	if prefs.MaxLineLength <= 0 {
		prefs.MaxLineLength = defaults.MaxLineLength
	}
	if prefs.LinesPerSubtitle <= 0 {
		prefs.LinesPerSubtitle = defaults.LinesPerSubtitle
	}
	if prefs.AlignmentChunkMinutes <= 0 {
		prefs.AlignmentChunkMinutes = defaults.AlignmentChunkMinutes
	}
	return prefs
}

func defaultPreferences() RunPreferences {
	return RunPreferences{
		MaxLineLength:         42,
		LinesPerSubtitle:      2,
		AlignmentChunkMinutes: 5,
	}
}

func fingerprint(request StartRequest) string {
	data, _ := json.Marshal(request)
	return string(data)
}

func loadTranscript(path string) (transcriptArtifact, error) {
	var artifact transcriptArtifact
	if err := readJSON(path, &artifact); err != nil {
		return transcriptArtifact{}, err
	}
	return artifact, nil
}

func loadWords(path string) ([]WordTimestamp, error) {
	var words []WordTimestamp
	if err := readJSON(path, &words); err != nil {
		return nil, err
	}
	return words, nil
}

func offsetWords(words []WordTimestamp, offsetMS int) []WordTimestamp {
	cloned := make([]WordTimestamp, 0, len(words))
	for _, word := range words {
		cloned = append(cloned, WordTimestamp{
			Text:       word.Text,
			StartMS:    word.StartMS + offsetMS,
			EndMS:      word.EndMS + offsetMS,
			Confidence: word.Confidence,
		})
	}
	return cloned
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
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

func (s *Service) defaultProbeDuration(ctx context.Context, inputPath string) (time.Duration, error) {
	cmd := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v",
		"error",
		"-show_entries",
		"format=duration",
		"-of",
		"default=noprint_wrappers=1:nokey=1",
		inputPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return time.ParseDuration(strings.TrimSpace(string(output)) + "s")
}

func (s *Service) defaultSegmentMedia(ctx context.Context, inputPath string, outputPath string, start time.Duration, duration time.Duration) error {
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i",
		inputPath,
		"-ss",
		fmt.Sprintf("%.3f", start.Seconds()),
		"-t",
		fmt.Sprintf("%.3f", duration.Seconds()),
		"-c",
		"copy",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("ffmpeg could not create a media chunk: %s", message)
	}
	return nil
}

func (s *Service) defaultRunWorker(ctx context.Context, request asrruntime.WorkerRequest) (asrruntime.WorkerResponse, error) {
	return s.runtime.RunWorker(ctx, request)
}
