package main

import (
	"path/filepath"

	"ASRSubs/internal/intake"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) SelectMediaFile() (*intake.MediaMetadata, error) {
	defaultDirectory := ""
	store, err := a.requireSettingsStore()
	if err == nil {
		preferences, loadErr := store.Load()
		if loadErr == nil {
			defaultDirectory = preferences.Directories.LastOpenDirectory
		}
	}

	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Choose media",
		DefaultDirectory: defaultDirectory,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Media Files",
				Pattern:     "*.wav;*.mp3;*.m4a;*.aac;*.flac;*.ogg;*.opus;*.mp4;*.mov;*.m4v;*.mkv;*.avi;*.webm",
			},
		},
	})
	if err != nil {
		a.recordDiagnostic("error", "intake", "The media picker could not be opened.")
		return nil, err
	}

	if path == "" {
		return nil, nil
	}

	return a.processMediaPath(path, "browse")
}

func (a *App) ValidateMediaPath(path string) (*intake.MediaMetadata, error) {
	return a.processMediaPath(path, "drop")
}

func (a *App) processMediaPath(path string, source string) (*intake.MediaMetadata, error) {
	metadata, err := a.intake.ValidateMediaFile(path)
	if err != nil {
		a.recordDiagnostic("warning", "intake", err.Error())
		return nil, err
	}

	if store, storeErr := a.requireSettingsStore(); storeErr == nil {
		preferences, loadErr := store.Load()
		if loadErr == nil {
			preferences.Directories.LastOpenDirectory = filepath.Dir(metadata.Path)
			_, _ = store.Save(preferences)
		}
	}

	a.recordDiagnostic("info", "intake", "Media file accepted from "+source+".")
	return &metadata, nil
}
