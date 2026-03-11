package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"ASRSubs/internal/settings"
	"ASRSubs/internal/transcription"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func TestSaveSubtitleDraftSavesValidatedTextAndPersistsDirectory(t *testing.T) {
	store := settings.NewStoreAtPath(filepath.Join(t.TempDir(), "settings.json"))
	if _, err := store.Save(settings.DefaultPreferences()); err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	app := &App{
		settings: store,
	}
	app.transcriptionState.draft = transcription.SubtitleDraft{
		SuggestedFilename: "clip.srt",
		SourceFilePath:    "/tmp/source/clip.wav",
		SourceFileName:    "clip.wav",
	}

	savePath := filepath.Join(t.TempDir(), "exports", "clip.srt")
	restore := stubSaveFileDialog(t, func(_ context.Context, options wailsruntime.SaveDialogOptions) (string, error) {
		if options.DefaultFilename != "clip.srt" {
			t.Fatalf("expected suggested filename, got %s", options.DefaultFilename)
		}
		return savePath, nil
	})
	defer restore()

	result, err := app.SaveSubtitleDraft(SubtitleSaveRequest{
		Text: "1\n00:00:00,000 --> 00:00:01,500\nhello\n",
	})
	if err != nil {
		t.Fatalf("save subtitle draft: %v", err)
	}

	if result.Status != "saved" {
		t.Fatalf("expected saved status, got %s", result.Status)
	}

	data, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("read saved subtitle: %v", err)
	}
	if string(data) != "1\n00:00:00,000 --> 00:00:01,500\nhello\n" {
		t.Fatalf("unexpected subtitle content: %q", string(data))
	}

	preferences, err := store.Load()
	if err != nil {
		t.Fatalf("reload settings: %v", err)
	}
	if preferences.Directories.LastSaveDirectory != filepath.Dir(savePath) {
		t.Fatalf("expected last save directory to persist, got %s", preferences.Directories.LastSaveDirectory)
	}
}

func TestSaveSubtitleDraftReturnsValidationIssueWithoutOpeningDialog(t *testing.T) {
	app := &App{}

	dialogOpened := false
	restore := stubSaveFileDialog(t, func(_ context.Context, options wailsruntime.SaveDialogOptions) (string, error) {
		dialogOpened = true
		return "", nil
	})
	defer restore()

	result, err := app.SaveSubtitleDraft(SubtitleSaveRequest{
		Text: "1\n00:00:01,000 --> 00:00:00,500\nbackwards\n",
	})
	if err != nil {
		t.Fatalf("save subtitle draft: %v", err)
	}
	if dialogOpened {
		t.Fatal("expected validation failure before dialog opens")
	}
	if result.Status != "invalid" {
		t.Fatalf("expected invalid status, got %s", result.Status)
	}
	if result.ValidationIssue == nil || result.ValidationIssue.Line != 2 {
		t.Fatalf("expected validation issue on line 2, got %#v", result.ValidationIssue)
	}
}

func TestConfirmDiscardSubtitleDraftTreatsSecondaryActionAsKeepEditing(t *testing.T) {
	app := &App{}

	restore := stubMessageDialog(t, func(_ context.Context, options wailsruntime.MessageDialogOptions) (string, error) {
		if options.Type != wailsruntime.QuestionDialog {
			t.Fatalf("expected question dialog, got %s", options.Type)
		}
		return "Keep editing", nil
	})
	defer restore()

	confirmed, err := app.ConfirmDiscardSubtitleDraft()
	if err != nil {
		t.Fatalf("confirm discard: %v", err)
	}
	if confirmed {
		t.Fatal("expected keep editing response to return false")
	}
}

func stubSaveFileDialog(
	t *testing.T,
	fn func(ctx context.Context, options wailsruntime.SaveDialogOptions) (string, error),
) func() {
	t.Helper()

	previous := saveFileDialog
	saveFileDialog = fn
	return func() {
		saveFileDialog = previous
	}
}

func stubMessageDialog(
	t *testing.T,
	fn func(ctx context.Context, options wailsruntime.MessageDialogOptions) (string, error),
) func() {
	t.Helper()

	previous := messageDialog
	messageDialog = fn
	return func() {
		messageDialog = previous
	}
}
