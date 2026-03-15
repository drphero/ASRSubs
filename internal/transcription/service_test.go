package transcription

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"ASRSubs/internal/models"
	asrruntime "ASRSubs/internal/runtime"
)

func TestTranscriptionRunsShortPipelineWithAlignmentAndSubtitles(t *testing.T) {
	harness := newTestHarness(t, testServiceConfig{})
	stages := []string{}
	downloadTargets := []string{}

	err := harness.service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "clip.wav"),
		ModelID:   "Qwen3-ASR-1.7B",
	}, func(snapshot Snapshot) {
		stages = append(stages, snapshot.Stage)
		if snapshot.Stage == StageDownloading {
			downloadTargets = append(downloadTargets, snapshot.DownloadTarget)
		}
	})
	if err != nil {
		t.Fatalf("start transcription: %v", err)
	}

	expected := []string{StagePreparingMedia, StageDownloading, StageDownloading, StageTranscribing, StageAligning, StageBuildingSubtitles}
	if len(stages) != len(expected) {
		t.Fatalf("unexpected stages: %v", stages)
	}
	for index, stage := range expected {
		if stages[index] != stage {
			t.Fatalf("expected stage %s at %d, got %s", stage, index, stages[index])
		}
	}
	expectedTargets := []string{"Qwen3-ASR-1.7B", models.ForcedAlignerID}
	if len(downloadTargets) != len(expectedTargets) {
		t.Fatalf("unexpected download targets: %v", downloadTargets)
	}
	for index, target := range expectedTargets {
		if downloadTargets[index] != target {
			t.Fatalf("expected download target %s at %d, got %s", target, index, downloadTargets[index])
		}
	}

	run := harness.service.lastRun
	if run == nil {
		t.Fatal("expected run state to be stored")
	}
	if !fileExists(run.TranscriptPath) {
		t.Fatalf("expected transcript artifact at %s", run.TranscriptPath)
	}
	if !fileExists(run.AlignmentPath) {
		t.Fatalf("expected alignment artifact at %s", run.AlignmentPath)
	}
	if !fileExists(run.TimelinePath) {
		t.Fatalf("expected timeline artifact at %s", run.TimelinePath)
	}
}

func TestTranscriptionDownloadsInternalAlignerBeforeAlignment(t *testing.T) {
	rootDir := t.TempDir()
	modelService := models.NewServiceAtRoot(rootDir, nil, models.WithDownloader(func(_ context.Context, model models.ModelDescriptor, destination string) error {
		return os.WriteFile(filepath.Join(destination, "weights.bin"), []byte(model.ID), 0o644)
	}))

	harness := newTestHarness(t, testServiceConfig{modelService: modelService})
	if err := harness.service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "clip.wav"),
		ModelID:   "Qwen3-ASR-0.6B",
	}, func(Snapshot) {}); err != nil {
		t.Fatalf("start transcription: %v", err)
	}

	alignerStatus, err := modelService.GetModelState(models.ForcedAlignerID)
	if err != nil {
		t.Fatalf("get aligner state: %v", err)
	}
	if alignerStatus.State != models.StateReady {
		t.Fatalf("expected internal aligner to be ready, got %s", alignerStatus.State)
	}

	snapshot := modelService.Snapshot()
	if len(snapshot.Models) != 2 {
		t.Fatalf("expected only selectable models in snapshot, got %d", len(snapshot.Models))
	}
}

func TestTranscriptionShowsAlignerAsDownloadTargetWhenOnlyAlignerIsMissing(t *testing.T) {
	rootDir := t.TempDir()
	modelService := models.NewServiceAtRoot(rootDir, nil, models.WithDownloader(func(_ context.Context, model models.ModelDescriptor, destination string) error {
		return os.WriteFile(filepath.Join(destination, "weights.bin"), []byte(model.ID), 0o644)
	}))
	waitForModelReady(t, modelService, "Qwen3-ASR-1.7B")

	harness := newTestHarness(t, testServiceConfig{modelService: modelService})
	downloadTargets := []string{}

	err := harness.service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "clip.wav"),
		ModelID:   "Qwen3-ASR-1.7B",
	}, func(snapshot Snapshot) {
		if snapshot.Stage == StageDownloading {
			downloadTargets = append(downloadTargets, snapshot.DownloadTarget)
		}
	})
	if err != nil {
		t.Fatalf("start transcription: %v", err)
	}

	expectedTargets := []string{models.ForcedAlignerID}
	if len(downloadTargets) != len(expectedTargets) {
		t.Fatalf("unexpected download targets: %v", downloadTargets)
	}
	if downloadTargets[0] != expectedTargets[0] {
		t.Fatalf("expected aligner download target, got %s", downloadTargets[0])
	}
}

func TestTranscriptionRetryResumesFromAlignmentArtifacts(t *testing.T) {
	harness := newTestHarness(t, testServiceConfig{
		failAlignOnce: true,
	})

	err := harness.service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "clip.wav"),
		ModelID:   "Qwen3-ASR-1.7B",
	}, func(Snapshot) {})
	if err == nil {
		t.Fatal("expected alignment failure")
	}

	failure, ok := err.(*Failure)
	if !ok {
		t.Fatalf("expected Failure, got %T", err)
	}
	if failure.Stage != StageAligning {
		t.Fatalf("expected aligning stage, got %s", failure.Stage)
	}
	if !fileExists(harness.service.lastRun.TranscriptPath) {
		t.Fatal("expected transcript artifact to survive failed alignment")
	}

	before := harness.tracker.calls["transcribe"]
	err = harness.service.Start(context.Background(), harness.service.lastRun.Request, func(Snapshot) {})
	if err != nil {
		t.Fatalf("retry transcription: %v", err)
	}
	if harness.tracker.calls["transcribe"] != before {
		t.Fatal("expected retry to reuse saved transcript artifact")
	}
}

func TestTranscriptionChunksLongMediaAndMergesOffsets(t *testing.T) {
	harness := newTestHarness(t, testServiceConfig{
		duration: 12 * time.Minute,
	})

	partCounts := []int{}
	err := harness.service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "long.wav"),
		ModelID:   "Qwen3-ASR-1.7B",
	}, func(snapshot Snapshot) {
		if snapshot.PartCount > 0 {
			partCounts = append(partCounts, snapshot.PartCount)
		}
	})
	if err != nil {
		t.Fatalf("start transcription: %v", err)
	}

	if len(harness.service.lastRun.ChunkPlan) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(harness.service.lastRun.ChunkPlan))
	}

	var timeline Timeline
	if err := readJSON(harness.service.lastRun.TimelinePath, &timeline); err != nil {
		t.Fatalf("read timeline: %v", err)
	}
	if len(timeline.Words) == 0 {
		t.Fatal("expected merged words")
	}
	if timeline.Words[len(timeline.Words)-1].StartMS < 10*60*1000 {
		t.Fatalf("expected merged offsets to include later chunks, got %d", timeline.Words[len(timeline.Words)-1].StartMS)
	}
	if len(partCounts) == 0 || partCounts[0] != 3 {
		t.Fatalf("expected chunk progress to include 3 parts, got %v", partCounts)
	}
}

func TestSRTSerializeUsesDeterministicOrderingAndFormatting(t *testing.T) {
	text := SerializeSRT([]SubtitleSegment{
		{
			Index:   99,
			StartMS: 0,
			EndMS:   1750,
			Lines:   []string{"hello world"},
		},
		{
			Index:   3,
			StartMS: 2000,
			EndMS:   4005,
			Lines:   []string{"second line", "wrapped"},
		},
	})

	expected := "1\n00:00:00,000 --> 00:00:01,750\nhello world\n\n2\n00:00:02,000 --> 00:00:04,005\nsecond line\nwrapped\n"
	if text != expected {
		t.Fatalf("unexpected srt output:\n%s", text)
	}
}

func TestSRTSerializeRepairsZeroDurationCue(t *testing.T) {
	text := SerializeSRT([]SubtitleSegment{
		{
			StartMS: 1000,
			EndMS:   1000,
			Lines:   []string{"hello"},
		},
	})

	expected := "1\n00:00:01,000 --> 00:00:01,050\nhello\n"
	if text != expected {
		t.Fatalf("unexpected repaired srt output:\n%s", text)
	}
}

func TestSRTSerializeRepairsBackwardsCueAndShiftsFollowingCue(t *testing.T) {
	text := SerializeSRT([]SubtitleSegment{
		{
			StartMS: 1000,
			EndMS:   900,
			Lines:   []string{"alpha"},
		},
		{
			StartMS: 1030,
			EndMS:   1130,
			Lines:   []string{"beta"},
		},
	})

	expected := "1\n00:00:01,000 --> 00:00:01,050\nalpha\n\n2\n00:00:01,050 --> 00:00:01,150\nbeta\n"
	if text != expected {
		t.Fatalf("unexpected repaired and shifted srt output:\n%s", text)
	}
}

func TestSRTSerializeRepairsConsecutiveZeroDurationCues(t *testing.T) {
	text := SerializeSRT([]SubtitleSegment{
		{
			StartMS: 1000,
			EndMS:   1000,
			Lines:   []string{"alpha"},
		},
		{
			StartMS: 1025,
			EndMS:   1025,
			Lines:   []string{"beta"},
		},
		{
			StartMS: 1030,
			EndMS:   1030,
			Lines:   []string{"gamma"},
		},
	})

	expected := "1\n00:00:01,000 --> 00:00:01,050\nalpha\n\n2\n00:00:01,050 --> 00:00:01,100\nbeta\n\n3\n00:00:01,100 --> 00:00:01,150\ngamma\n"
	if text != expected {
		t.Fatalf("unexpected repaired consecutive srt output:\n%s", text)
	}
}

func TestValidateSRTRejectsMissingBlankLineAndBackwardsTime(t *testing.T) {
	missingGap := "1\n00:00:00,000 --> 00:00:01,000\nhello\n2\n00:00:01,100 --> 00:00:02,000\nworld\n"
	issue := ValidateSRT(missingGap)
	if issue == nil || issue.Line != 4 {
		t.Fatalf("expected blank-line issue on line 4, got %#v", issue)
	}

	backwards := "1\n00:00:02,000 --> 00:00:01,500\nhello\n"
	issue = ValidateSRT(backwards)
	if issue == nil || issue.Line != 2 {
		t.Fatalf("expected backwards-time issue on line 2, got %#v", issue)
	}
}

func TestSubtitleDraftLoadsSerializedTimelineFromLatestRun(t *testing.T) {
	harness := newTestHarness(t, testServiceConfig{})

	err := harness.service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "clip.wav"),
		ModelID:   "Qwen3-ASR-1.7B",
	}, func(Snapshot) {})
	if err != nil {
		t.Fatalf("start transcription: %v", err)
	}

	draft, err := harness.service.GetLatestSubtitleDraft()
	if err != nil {
		t.Fatalf("get latest subtitle draft: %v", err)
	}
	if draft.SuggestedFilename != "clip.srt" {
		t.Fatalf("expected clip.srt, got %s", draft.SuggestedFilename)
	}
	if !strings.Contains(draft.Text, "00:00:00,000 --> 00:00:01,240") {
		t.Fatalf("expected serialized timestamps, got %s", draft.Text)
	}
}

func TestSubtitleDraftRepairsInvalidTimelineCueTimings(t *testing.T) {
	harness := newTestHarness(t, testServiceConfig{})

	err := harness.service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "clip.wav"),
		ModelID:   "Qwen3-ASR-1.7B",
	}, func(Snapshot) {})
	if err != nil {
		t.Fatalf("start transcription: %v", err)
	}

	var timeline Timeline
	if err := readJSON(harness.service.lastRun.TimelinePath, &timeline); err != nil {
		t.Fatalf("read timeline: %v", err)
	}

	timeline.Subtitles = []SubtitleSegment{
		{
			StartMS: 1000,
			EndMS:   1000,
			Lines:   []string{"alpha"},
		},
		{
			StartMS: 1020,
			EndMS:   1020,
			Lines:   []string{"beta"},
		},
	}
	if err := writeJSON(harness.service.lastRun.TimelinePath, timeline); err != nil {
		t.Fatalf("write timeline: %v", err)
	}

	draft, err := harness.service.GetLatestSubtitleDraft()
	if err != nil {
		t.Fatalf("get latest subtitle draft: %v", err)
	}
	if issue := ValidateSRT(draft.Text); issue != nil {
		t.Fatalf("expected repaired draft to validate, got %#v\n%s", issue, draft.Text)
	}

	expected := "1\n00:00:01,000 --> 00:00:01,050\nalpha\n\n2\n00:00:01,050 --> 00:00:01,100\nbeta\n"
	if draft.Text != expected {
		t.Fatalf("unexpected repaired draft text:\n%s", draft.Text)
	}
}

func TestTranscriptionChunkFailureRetriesOnceBeforeReturningFailure(t *testing.T) {
	harness := newTestHarness(t, testServiceConfig{
		duration:            8 * time.Minute,
		failChunkAlignments: map[int]int{2: 2},
	})

	err := harness.service.Start(context.Background(), StartRequest{
		MediaPath: filepath.Join(t.TempDir(), "long.wav"),
		ModelID:   "Qwen3-ASR-1.7B",
	}, func(Snapshot) {})
	if err == nil {
		t.Fatal("expected chunk alignment failure")
	}

	failure, ok := err.(*Failure)
	if !ok {
		t.Fatalf("expected Failure, got %T", err)
	}
	if failure.PartIndex != 2 || failure.PartCount != 2 {
		t.Fatalf("expected failure on second chunk, got part %d/%d", failure.PartIndex, failure.PartCount)
	}
	if harness.tracker.calls["align"] != 3 {
		t.Fatalf("expected one retry for failing chunk, got %d align calls", harness.tracker.calls["align"])
	}
}

func TestResolveBinaryPathPrefersBundledResources(t *testing.T) {
	resourceRoot := t.TempDir()
	binDir := filepath.Join(resourceRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bundled bin dir: %v", err)
	}

	ffmpegName := "ffmpeg"
	if goruntime.GOOS == "windows" {
		ffmpegName += ".exe"
	}

	bundledPath := filepath.Join(binDir, ffmpegName)
	if err := os.WriteFile(bundledPath, []byte("stub"), 0o755); err != nil {
		t.Fatalf("write bundled ffmpeg: %v", err)
	}

	t.Setenv("ASRSUBS_RESOURCE_ROOT", resourceRoot)

	service := &Service{}
	resolved, err := service.resolveBinaryPath("ffmpeg")
	if err != nil {
		t.Fatalf("resolve binary path: %v", err)
	}
	if resolved != bundledPath {
		t.Fatalf("expected bundled ffmpeg path, got %s", resolved)
	}
}

func TestResolveBinaryPathFallsBackToPath(t *testing.T) {
	binDir := t.TempDir()
	ffprobeName := "ffprobe"
	if goruntime.GOOS == "windows" {
		ffprobeName += ".exe"
	}

	ffprobePath := filepath.Join(binDir, ffprobeName)
	if err := os.WriteFile(ffprobePath, []byte("stub"), 0o755); err != nil {
		t.Fatalf("write ffprobe stub: %v", err)
	}

	t.Setenv("ASRSUBS_RESOURCE_ROOT", "")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	service := &Service{}
	resolved, err := service.resolveBinaryPath("ffprobe")
	if err != nil {
		t.Fatalf("resolve binary path: %v", err)
	}
	if resolved != ffprobePath {
		t.Fatalf("expected PATH ffprobe path, got %s", resolved)
	}
}

type testServiceConfig struct {
	modelService        *models.Service
	duration            time.Duration
	failAlignOnce       bool
	failChunkAlignments map[int]int
	subtitleError       error
}

type testHarness struct {
	service *Service
	tracker *workerTracker
}

func newTestHarness(t *testing.T, cfg testServiceConfig) testHarness {
	t.Helper()

	modelService := cfg.modelService
	if modelService == nil {
		modelService = models.NewServiceAtRoot(t.TempDir(), nil, models.WithDownloader(func(_ context.Context, model models.ModelDescriptor, destination string) error {
			return os.WriteFile(filepath.Join(destination, "weights.bin"), []byte(model.ID), 0o644)
		}))
	}

	tracker := &workerTracker{
		calls:               map[string]int{},
		failAlignOnce:       cfg.failAlignOnce,
		failChunkAlignments: cfg.failChunkAlignments,
	}

	service := NewService(
		newFakeRuntimeService(t),
		modelService,
		WithMediaPreparer(func(_ context.Context, inputPath string, outputPath string) error {
			return os.WriteFile(outputPath, []byte(inputPath), 0o644)
		}),
		WithDurationProber(func(context.Context, string) (time.Duration, error) {
			if cfg.duration > 0 {
				return cfg.duration, nil
			}
			return 2 * time.Minute, nil
		}),
		WithMediaSegmenter(func(_ context.Context, _ string, outputPath string, _ time.Duration, _ time.Duration) error {
			return os.WriteFile(outputPath, []byte("chunk"), 0o644)
		}),
		WithWorkerRunner(tracker.run),
		WithSubtitleBuilder(func(words []WordTimestamp, prefs RunPreferences) ([]SubtitleSegment, error) {
			if cfg.subtitleError != nil {
				return nil, cfg.subtitleError
			}
			return BuildSubtitles(words, prefs)
		}),
		WithTempDir(t.TempDir()),
	)

	return testHarness{
		service: service,
		tracker: tracker,
	}
}

type workerTracker struct {
	calls               map[string]int
	failAlignOnce       bool
	failChunkAlignments map[int]int
}

func waitForModelReady(t *testing.T, service *models.Service, modelID string) {
	t.Helper()

	if _, err := service.StartDownload(modelID); err != nil {
		t.Fatalf("start model download: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, err := service.GetModelState(modelID)
		if err != nil {
			t.Fatalf("get model state: %v", err)
		}
		if status.State == models.StateReady {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	status, err := service.GetModelState(modelID)
	if err != nil {
		t.Fatalf("get model state: %v", err)
	}
	t.Fatalf("timed out waiting for %s to become ready, got %s", modelID, status.State)
}

func (w *workerTracker) run(_ context.Context, request asrruntime.WorkerRequest) (asrruntime.WorkerResponse, error) {
	w.calls[request.Command]++

	switch request.Command {
	case "transcribe":
		return asrruntime.WorkerResponse{
			OK:      true,
			Command: "transcribe",
			Details: mustJSON(asrruntime.TranscriptPayload{
				Text:     "alpha beta gamma",
				Language: "en",
				Words: []asrruntime.TranscriptToken{
					{Text: "alpha"},
					{Text: "beta"},
					{Text: "gamma"},
				},
			}),
		}, nil
	case "align":
		if w.failAlignOnce {
			w.failAlignOnce = false
			return asrruntime.WorkerResponse{}, errors.New("aligner crashed")
		}

		chunkIndex := 0
		base := filepath.Base(request.AudioPath)
		switch {
		case strings.Contains(base, "chunk-01"):
			chunkIndex = 1
		case strings.Contains(base, "chunk-02"):
			chunkIndex = 2
		case strings.Contains(base, "chunk-03"):
			chunkIndex = 3
		}

		if remaining := w.failChunkAlignments[chunkIndex]; remaining > 0 {
			w.failChunkAlignments[chunkIndex] = remaining - 1
			return asrruntime.WorkerResponse{}, errors.New("chunk alignment failed")
		}

		words := []asrruntime.AlignedWord{
			{Text: "alpha", StartMS: 0, EndMS: 320, Confidence: 0.99},
			{Text: "beta", StartMS: 400, EndMS: 760, Confidence: 0.99},
			{Text: "gamma", StartMS: 840, EndMS: 1240, Confidence: 0.99},
		}
		return asrruntime.WorkerResponse{
			OK:      true,
			Command: "align",
			Details: mustJSON(asrruntime.AlignmentPayload{Words: words}),
		}, nil
	default:
		return asrruntime.WorkerResponse{}, errors.New("unsupported worker command")
	}
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
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
