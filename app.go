package main

import (
	"context"

	"ASRSubs/internal/intake"
	"ASRSubs/internal/settings"
)

type App struct {
	ctx      context.Context
	intake   *intake.Service
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

	a.settings = store
	a.recordDiagnostic("info", "app", "The shell is ready for a media file.")
}
