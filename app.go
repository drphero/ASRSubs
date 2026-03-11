package main

import (
	"context"

	"ASRSubs/internal/intake"
	asrruntime "ASRSubs/internal/runtime"
	"ASRSubs/internal/settings"
)

type App struct {
	ctx      context.Context
	intake   *intake.Service
	runtime  *asrruntime.Service
	settings *settings.Store

	diagnostics diagnosticsState
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
	a.recordDiagnostic("info", "app", "The shell is ready for a media file.")
}
