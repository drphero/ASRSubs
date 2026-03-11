package settings

import (
	"path/filepath"
	"testing"
)

func TestSettingsLoadDefaults(t *testing.T) {
	t.Parallel()

	store := NewStoreAtPath(filepath.Join(t.TempDir(), "settings.json"))
	preferences, err := store.Load()
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}

	if preferences.Theme != ThemeDark {
		t.Fatalf("expected dark theme, got %s", preferences.Theme)
	}

	if preferences.Model != ModelLarge {
		t.Fatalf("expected default model, got %s", preferences.Model)
	}
}

func TestSettingsPersistPreferences(t *testing.T) {
	t.Parallel()

	store := NewStoreAtPath(filepath.Join(t.TempDir(), "settings.json"))
	saved, err := store.Save(Preferences{
		Version: 1,
		Model:   ModelLarge,
		Theme:   ThemeLight,
		Output: OutputPreferences{
			MaxLineLength:    36,
			LinesPerSubtitle: 3,
		},
		Directories: DirectoryPreferences{
			LastOpenDirectory: "/tmp/input",
			LastSaveDirectory: "/tmp/output",
		},
		Processing: ProcessingPreferences{
			AlignmentChunkMinutes: 4,
			OneWordPerSubtitle:    true,
		},
	})
	if err != nil {
		t.Fatalf("save preferences: %v", err)
	}

	if saved.Theme != ThemeLight {
		t.Fatalf("expected light theme after save, got %s", saved.Theme)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("reload preferences: %v", err)
	}

	if loaded.Directories.LastOpenDirectory != "/tmp/input" {
		t.Fatalf("expected directory to persist, got %s", loaded.Directories.LastOpenDirectory)
	}

	if loaded.Processing.AlignmentChunkMinutes != 4 {
		t.Fatalf("expected chunk minutes to persist, got %d", loaded.Processing.AlignmentChunkMinutes)
	}

	if !loaded.Processing.OneWordPerSubtitle {
		t.Fatal("expected one-word subtitle mode to persist")
	}
}
