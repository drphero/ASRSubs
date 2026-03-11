package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"ASRSubs/internal/transcription"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type transcriptionState struct {
	mu          sync.RWMutex
	lastRequest transcription.StartRequest
	snapshot    transcription.Snapshot
}

func (a *App) GetTranscriptionSnapshot() transcription.Snapshot {
	a.transcriptionState.mu.RLock()
	defer a.transcriptionState.mu.RUnlock()
	return a.transcriptionState.snapshot
}

func (a *App) StartTranscription(request transcription.StartRequest) (transcription.Snapshot, error) {
	return a.beginTranscription(request)
}

func (a *App) RetryTranscription() (transcription.Snapshot, error) {
	a.transcriptionState.mu.RLock()
	request := a.transcriptionState.lastRequest
	a.transcriptionState.mu.RUnlock()

	if request.MediaPath == "" || request.ModelID == "" {
		return transcription.Snapshot{}, fmt.Errorf("no transcription is available to retry")
	}

	return a.beginTranscription(request)
}

func (a *App) beginTranscription(request transcription.StartRequest) (transcription.Snapshot, error) {
	service, err := a.requireTranscriptionService()
	if err != nil {
		return transcription.Snapshot{}, err
	}

	if request.MediaPath == "" || request.ModelID == "" {
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
		FilePath: request.MediaPath,
		FileName: filepath.Base(request.MediaPath),
		ModelID:  request.ModelID,
		Stage:    transcription.StagePreparingMedia,
	}
	a.transcriptionState.lastRequest = request
	a.transcriptionState.snapshot = snapshot
	a.transcriptionState.mu.Unlock()

	a.emitTranscriptionSnapshot(snapshot)

	go func() {
		ctx := context.Background()
		runErr := service.Start(ctx, request, func(update transcription.Snapshot) {
			a.transcriptionState.mu.Lock()
			a.transcriptionState.snapshot = update
			a.transcriptionState.mu.Unlock()
			a.emitTranscriptionSnapshot(update)
		})

		if runErr == nil {
			a.recordDiagnostic("info", "transcription", "Local transcription finished.")
			a.transcriptionState.mu.Lock()
			a.transcriptionState.snapshot = transcription.Snapshot{
				Active:   false,
				CanRetry: false,
				FilePath: request.MediaPath,
				FileName: filepath.Base(request.MediaPath),
				ModelID:  request.ModelID,
			}
			completed := a.transcriptionState.snapshot
			a.transcriptionState.mu.Unlock()
			a.emitTranscriptionSnapshot(completed)
			return
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
		a.transcriptionState.snapshot = transcription.Snapshot{
			Active:         false,
			CanRetry:       true,
			FilePath:       request.MediaPath,
			FileName:       filepath.Base(request.MediaPath),
			ModelID:        request.ModelID,
			FailureSummary: summary,
		}
		failed := a.transcriptionState.snapshot
		a.transcriptionState.mu.Unlock()
		a.emitTranscriptionSnapshot(failed)
	}()

	return snapshot, nil
}

func (a *App) requireTranscriptionService() (*transcription.Service, error) {
	if a.transcription == nil {
		return nil, fmt.Errorf("transcription service is not ready")
	}

	return a.transcription, nil
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
