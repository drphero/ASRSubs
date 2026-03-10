package main

import (
	"fmt"

	"ASRSubs/internal/settings"
)

func (a *App) LoadPreferences() (settings.Preferences, error) {
	store, err := a.requireSettingsStore()
	if err != nil {
		return settings.DefaultPreferences(), err
	}

	preferences, err := store.Load()
	if err != nil {
		a.recordDiagnostic("error", "settings", "Saved settings could not be loaded.")
		return settings.DefaultPreferences(), err
	}

	a.recordDiagnostic("info", "settings", "Saved settings were loaded.")
	return preferences, nil
}

func (a *App) UpdatePreferences(next settings.Preferences) (settings.Preferences, error) {
	store, err := a.requireSettingsStore()
	if err != nil {
		return settings.DefaultPreferences(), err
	}

	saved, err := store.Save(next)
	if err != nil {
		a.recordDiagnostic("error", "settings", "Settings could not be saved.")
		return settings.DefaultPreferences(), err
	}

	a.recordDiagnostic("info", "settings", "Settings were updated.")
	return saved, nil
}

func (a *App) requireSettingsStore() (*settings.Store, error) {
	if a.settings == nil {
		return nil, fmt.Errorf("settings storage is not ready")
	}

	return a.settings, nil
}
