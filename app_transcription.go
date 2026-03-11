package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"ASRSubs/internal/transcription"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type transcriptionState struct {
	mu          sync.RWMutex
	lastRequest transcription.StartRequest
	snapshot    transcription.Snapshot
	draft       transcription.SubtitleDraft
}

type transcriptionRequest struct {
	MediaPath string `json:"mediaPath"`
	ModelID   string `json:"modelID"`
}

type SubtitleSaveRequest struct {
	Text              string `json:"text"`
	SuggestedFilename string `json:"suggestedFilename"`
}

type SubtitleSaveResult struct {
	Status          string                         `json:"status"`
	Path            string                         `json:"path,omitempty"`
	FileName        string                         `json:"fileName,omitempty"`
	ValidationIssue *transcription.ValidationIssue `json:"validationIssue,omitempty"`
}

var saveFileDialog = wailsruntime.SaveFileDialog
var messageDialog = wailsruntime.MessageDialog

func (a *App) GetTranscriptionSnapshot() transcription.Snapshot {
	a.transcriptionState.mu.RLock()
	defer a.transcriptionState.mu.RUnlock()
	return a.transcriptionState.snapshot
}

func (a *App) GetSubtitleDraft() (transcription.SubtitleDraft, error) {
	a.transcriptionState.mu.RLock()
	defer a.transcriptionState.mu.RUnlock()

	if a.transcriptionState.draft.Text == "" {
		return transcription.SubtitleDraft{}, fmt.Errorf("no subtitle draft is available")
	}

	return a.transcriptionState.draft, nil
}

func (a *App) StartTranscription(request transcriptionRequest) (transcription.Snapshot, error) {
	return a.beginTranscription(request)
}

func (a *App) RetryTranscription() (transcription.Snapshot, error) {
	a.transcriptionState.mu.RLock()
	request := a.transcriptionState.lastRequest
	a.transcriptionState.mu.RUnlock()

	if request.MediaPath == "" || request.ModelID == "" {
		return transcription.Snapshot{}, fmt.Errorf("no transcription is available to retry")
	}

	return a.beginTranscriptionWithRequest(request)
}

func (a *App) beginTranscription(request transcriptionRequest) (transcription.Snapshot, error) {
	serviceRequest := transcription.StartRequest{
		MediaPath: request.MediaPath,
		ModelID:   request.ModelID,
	}
	if a.settings != nil {
		preferences, prefErr := a.settings.Load()
		if prefErr == nil {
			serviceRequest.Preferences = transcription.RunPreferences{
				MaxLineLength:         preferences.Output.MaxLineLength,
				LinesPerSubtitle:      preferences.Output.LinesPerSubtitle,
				AlignmentChunkMinutes: preferences.Processing.AlignmentChunkMinutes,
				OneWordPerSubtitle:    preferences.Processing.OneWordPerSubtitle,
			}
		}
	}

	return a.beginTranscriptionWithRequest(serviceRequest)
}

func (a *App) beginTranscriptionWithRequest(serviceRequest transcription.StartRequest) (transcription.Snapshot, error) {
	service, err := a.requireTranscriptionService()
	if err != nil {
		return transcription.Snapshot{}, err
	}

	if serviceRequest.MediaPath == "" || serviceRequest.ModelID == "" {
		return transcription.Snapshot{}, fmt.Errorf("transcription requires a media file and selected model")
	}

	a.transcriptionState.mu.Lock()
	if a.transcriptionState.snapshot.Active {
		a.transcriptionState.mu.Unlock()
		return transcription.Snapshot{}, fmt.Errorf("a transcription is already in progress")
	}

	snapshot := transcription.Snapshot{
		Active:   true,
		CanRetry: false,
		FilePath: serviceRequest.MediaPath,
		FileName: filepath.Base(serviceRequest.MediaPath),
		ModelID:  serviceRequest.ModelID,
		Stage:    transcription.StagePreparingMedia,
	}
	a.transcriptionState.lastRequest = serviceRequest
	a.transcriptionState.snapshot = snapshot
	a.transcriptionState.draft = transcription.SubtitleDraft{}
	a.transcriptionState.mu.Unlock()

	a.emitTranscriptionSnapshot(snapshot)

	go func() {
		ctx := context.Background()
		runErr := service.Start(ctx, serviceRequest, func(update transcription.Snapshot) {
			a.transcriptionState.mu.Lock()
			a.transcriptionState.snapshot = update
			a.transcriptionState.mu.Unlock()
			a.emitTranscriptionSnapshot(update)
		})

		if runErr == nil {
			draft, draftErr := service.GetLatestSubtitleDraft()
			if draftErr != nil {
				runErr = draftErr
			} else {
				a.recordDiagnostic("info", "transcription", "Local transcription finished.")
				a.transcriptionState.mu.Lock()
				a.transcriptionState.draft = draft
				a.transcriptionState.snapshot = transcription.Snapshot{
					Active:   false,
					CanRetry: false,
					FilePath: serviceRequest.MediaPath,
					FileName: filepath.Base(serviceRequest.MediaPath),
					ModelID:  serviceRequest.ModelID,
				}
				completed := a.transcriptionState.snapshot
				a.transcriptionState.mu.Unlock()
				a.emitTranscriptionSnapshot(completed)
				return
			}
		}

		summary := "Transcription could not start."
		detail := runErr.Error()
		var failure *transcription.Failure
		if ok := errorAsTranscriptionFailure(runErr, &failure); ok {
			summary = failure.Summary
			detail = failure.Detail
		}

		a.recordDiagnostic("error", "transcription", detail)
		a.transcriptionState.mu.Lock()
		a.transcriptionState.draft = transcription.SubtitleDraft{}
		a.transcriptionState.snapshot = transcription.Snapshot{
			Active:         false,
			CanRetry:       true,
			FilePath:       serviceRequest.MediaPath,
			FileName:       filepath.Base(serviceRequest.MediaPath),
			ModelID:        serviceRequest.ModelID,
			FailureSummary: summary,
		}
		if failure != nil {
			a.transcriptionState.snapshot.FailedStage = failure.Stage
			a.transcriptionState.snapshot.PartIndex = failure.PartIndex
			a.transcriptionState.snapshot.PartCount = failure.PartCount
			a.transcriptionState.snapshot.CanRetry = failure.Retryable
		}
		failed := a.transcriptionState.snapshot
		a.transcriptionState.mu.Unlock()
		a.emitTranscriptionSnapshot(failed)
	}()

	return snapshot, nil
}

func (a *App) SaveSubtitleDraft(request SubtitleSaveRequest) (SubtitleSaveResult, error) {
	text := request.Text
	if issue := transcription.ValidateSRT(text); issue != nil {
		return SubtitleSaveResult{
			Status:          "invalid",
			ValidationIssue: issue,
		}, nil
	}

	a.transcriptionState.mu.RLock()
	draft := a.transcriptionState.draft
	a.transcriptionState.mu.RUnlock()

	defaultFilename := request.SuggestedFilename
	if defaultFilename == "" {
		defaultFilename = draft.SuggestedFilename
	}
	if defaultFilename == "" {
		defaultFilename = transcription.DraftFilenameForMedia(draft.SourceFilePath)
	}

	defaultDirectory := a.defaultSaveDirectory(draft.SourceFilePath)
	path, err := saveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:                "Save subtitles",
		DefaultDirectory:     defaultDirectory,
		DefaultFilename:      defaultFilename,
		CanCreateDirectories: true,
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "SubRip Subtitle (*.srt)",
				Pattern:     "*.srt",
			},
		},
	})
	if err != nil {
		a.recordDiagnostic("error", "transcription", "The subtitle save dialog could not be opened.")
		return SubtitleSaveResult{}, err
	}
	if path == "" {
		return SubtitleSaveResult{Status: "canceled"}, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return SubtitleSaveResult{}, err
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return SubtitleSaveResult{}, err
	}

	if store, storeErr := a.requireSettingsStore(); storeErr == nil {
		if preferences, loadErr := store.Load(); loadErr == nil {
			preferences.Directories.LastSaveDirectory = filepath.Dir(path)
			_, _ = store.Save(preferences)
		}
	}

	a.recordDiagnostic("info", "transcription", "Subtitle draft saved to "+path+".")
	return SubtitleSaveResult{
		Status:   "saved",
		Path:     path,
		FileName: filepath.Base(path),
	}, nil
}

func (a *App) ConfirmDiscardSubtitleDraft() (bool, error) {
	choice, err := messageDialog(a.ctx, wailsruntime.MessageDialogOptions{
		Type:          wailsruntime.QuestionDialog,
		Title:         "Discard subtitle edits?",
		Message:       "Unsaved subtitle edits will be lost.",
		Buttons:       []string{"Keep editing", "Discard edits"},
		DefaultButton: "Keep editing",
		CancelButton:  "Keep editing",
	})
	if err != nil {
		return false, err
	}

	return choice == "Discard edits", nil
}

func (a *App) requireTranscriptionService() (*transcription.Service, error) {
	if a.transcription == nil {
		return nil, fmt.Errorf("transcription service is not ready")
	}

	return a.transcription, nil
}

func (a *App) defaultSaveDirectory(sourcePath string) string {
	if store, err := a.requireSettingsStore(); err == nil {
		if preferences, loadErr := store.Load(); loadErr == nil {
			candidate := preferences.Directories.LastSaveDirectory
			if candidate != "" {
				if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
					return candidate
				}
			}
		}
	}

	if sourcePath == "" {
		return ""
	}

	directory := filepath.Dir(sourcePath)
	if info, err := os.Stat(directory); err == nil && info.IsDir() {
		return directory
	}

	return ""
}

func (a *App) emitTranscriptionSnapshot(snapshot transcription.Snapshot) {
	if a.ctx == nil {
		return
	}

	wailsruntime.EventsEmit(a.ctx, "transcription:state", snapshot)
}

func errorAsTranscriptionFailure(err error, target **transcription.Failure) bool {
	failure, ok := err.(*transcription.Failure)
	if ok {
		*target = failure
	}
	return ok
}
