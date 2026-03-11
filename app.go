package main

import (
	"context"

	"ASRSubs/internal/intake"
	"ASRSubs/internal/models"
	asrruntime "ASRSubs/internal/runtime"
	"ASRSubs/internal/settings"
	"ASRSubs/internal/transcription"
)

type App struct {
	ctx           context.Context
	intake        *intake.Service
	models        *models.Service
	runtime       *asrruntime.Service
	settings      *settings.Store
	transcription *transcription.Service

	diagnostics        diagnosticsState
	transcriptionState transcriptionState
}

func NewApp() *App {
	return &App{
		intake: intake.NewService(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.initDiagnostics()

	store, err := settings.NewStore("ASRSubs")
	if err != nil {
		a.recordDiagnostic("error", "settings", "Settings storage could not be prepared.")
		return
	}

	runtimeService, err := asrruntime.NewService("ASRSubs")
	if err != nil {
		a.recordDiagnostic("error", "runtime", "Managed runtime storage could not be prepared.")
		return
	}

	a.settings = store
	a.runtime = runtimeService

	modelService, err := models.NewService("ASRSubs", runtimeService, models.WithStateEmitter(a.emitModelSnapshot))
	if err != nil {
		a.recordDiagnostic("error", "models", "Model storage could not be prepared.")
		return
	}

	a.models = modelService
	a.transcription = transcription.NewService(runtimeService, modelService)
	a.recordDiagnostic("info", "app", "The shell is ready for a media file.")
}
