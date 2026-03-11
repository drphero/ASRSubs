package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	ThemeDark  = "dark"
	ThemeLight = "light"
	ModelSmall = "Qwen3-ASR-0.6B"
	ModelLarge = "Qwen3-ASR-1.7B"
)

type Preferences struct {
	Version     int                   `json:"version"`
	Model       string                `json:"model"`
	Theme       string                `json:"theme"`
	Output      OutputPreferences     `json:"output"`
	Directories DirectoryPreferences  `json:"directories"`
	Processing  ProcessingPreferences `json:"processing"`
}

type OutputPreferences struct {
	MaxLineLength    int `json:"maxLineLength"`
	LinesPerSubtitle int `json:"linesPerSubtitle"`
}

type DirectoryPreferences struct {
	LastOpenDirectory string `json:"lastOpenDirectory"`
	LastSaveDirectory string `json:"lastSaveDirectory"`
}

type ProcessingPreferences struct {
	AlignmentChunkMinutes int `json:"alignmentChunkMinutes"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(appName string) (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	return &Store{
		path: filepath.Join(configDir, appName, "settings.json"),
	}, nil
}

func NewStoreAtPath(path string) *Store {
	return &Store{path: path}
}

func DefaultPreferences() Preferences {
	return Preferences{
		Version: 1,
		Model:   ModelLarge,
		Theme:   ThemeDark,
		Output: OutputPreferences{
			MaxLineLength:    42,
			LinesPerSubtitle: 2,
		},
		Directories: DirectoryPreferences{},
		Processing: ProcessingPreferences{
			AlignmentChunkMinutes: 5,
		},
	}
}

func (s *Store) Load() (Preferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaults := DefaultPreferences()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaults, nil
		}
		return defaults, err
	}

	var preferences Preferences
	if err := json.Unmarshal(data, &preferences); err != nil {
		return defaults, fmt.Errorf("decode settings: %w", err)
	}

	return sanitize(preferences), nil
}

func (s *Store) Save(preferences Preferences) (Preferences, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sanitized := sanitize(preferences)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return DefaultPreferences(), err
	}

	data, err := json.MarshalIndent(sanitized, "", "  ")
	if err != nil {
		return DefaultPreferences(), err
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return DefaultPreferences(), err
	}

	return sanitized, nil
}

func sanitize(preferences Preferences) Preferences {
	defaults := DefaultPreferences()

	preferences.Version = defaults.Version

	if preferences.Model != ModelSmall && preferences.Model != ModelLarge {
		preferences.Model = defaults.Model
	}

	if preferences.Theme != ThemeDark && preferences.Theme != ThemeLight {
		preferences.Theme = defaults.Theme
	}

	if preferences.Output.MaxLineLength <= 0 {
		preferences.Output.MaxLineLength = defaults.Output.MaxLineLength
	}
	if preferences.Output.LinesPerSubtitle <= 0 {
		preferences.Output.LinesPerSubtitle = defaults.Output.LinesPerSubtitle
	}
	if preferences.Processing.AlignmentChunkMinutes <= 0 {
		preferences.Processing.AlignmentChunkMinutes = defaults.Processing.AlignmentChunkMinutes
	}

	return preferences
}
