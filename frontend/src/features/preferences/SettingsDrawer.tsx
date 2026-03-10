import type { Preferences } from "../../lib/backend";

type SettingsDrawerProps = {
  error: string | null;
  onClose: () => void;
  onPreferencesChange: (preferences: Preferences) => void | Promise<unknown>;
  open: boolean;
  preferences: Preferences;
};

export function SettingsDrawer({
  error,
  onClose,
  onPreferencesChange,
  open,
  preferences,
}: SettingsDrawerProps) {
  if (!open) {
    return null;
  }

  return (
    <div className="overlay-shell" role="presentation">
      <button aria-label="Close settings" className="overlay-backdrop" onClick={onClose} type="button" />
      <aside aria-label="settings drawer" className="drawer-panel" role="dialog">
        <div className="drawer-header">
          <div>
            <p className="section-label">Preferences</p>
            <h2>Settings</h2>
          </div>
          <button className="ghost-action" onClick={onClose} type="button">
            Close
          </button>
        </div>

        <div className="drawer-group">
          <label className="field">
            <span>Transcription model</span>
            <select
              aria-label="Transcription model"
              onChange={(event) =>
                onPreferencesChange({
                  ...preferences,
                  model: event.target.value as Preferences["model"],
                })
              }
              value={preferences.model}
            >
              <option value="Qwen3-ASR-0.6B">Qwen3-ASR-0.6B</option>
              <option value="Qwen3-ASR-1.7B">Qwen3-ASR-1.7B</option>
            </select>
          </label>

          <label className="field">
            <span>Theme</span>
            <select
              aria-label="Theme"
              onChange={(event) =>
                onPreferencesChange({
                  ...preferences,
                  theme: event.target.value as Preferences["theme"],
                })
              }
              value={preferences.theme}
            >
              <option value="dark">Dark</option>
              <option value="light">Light</option>
            </select>
          </label>
        </div>

        <div className="drawer-group">
          <label className="field">
            <span>Max line length</span>
            <input
              aria-label="Max line length"
              min={20}
              onChange={(event) =>
                onPreferencesChange({
                  ...preferences,
                  output: {
                    ...preferences.output,
                    maxLineLength: Number(event.target.value),
                  },
                })
              }
              type="number"
              value={preferences.output.maxLineLength}
            />
          </label>

          <label className="field">
            <span>Lines per subtitle</span>
            <input
              aria-label="Lines per subtitle"
              min={1}
              onChange={(event) =>
                onPreferencesChange({
                  ...preferences,
                  output: {
                    ...preferences.output,
                    linesPerSubtitle: Number(event.target.value),
                  },
                })
              }
              type="number"
              value={preferences.output.linesPerSubtitle}
            />
          </label>

          <label className="field">
            <span>Alignment chunk size (minutes)</span>
            <input
              aria-label="Alignment chunk size (minutes)"
              min={1}
              onChange={(event) =>
                onPreferencesChange({
                  ...preferences,
                  processing: {
                    ...preferences.processing,
                    alignmentChunkMinutes: Number(event.target.value),
                  },
                })
              }
              type="number"
              value={preferences.processing.alignmentChunkMinutes}
            />
          </label>
        </div>

        <div className="drawer-group">
          <div className="field field-static">
            <span>Last media folder</span>
            <strong>{preferences.directories.lastOpenDirectory || "No folder yet"}</strong>
          </div>
          <div className="field field-static">
            <span>Last save folder</span>
            <strong>{preferences.directories.lastSaveDirectory || "No folder yet"}</strong>
          </div>
        </div>

        {error ? (
          <p className="inline-feedback inline-feedback-error" role="alert">
            {error}
          </p>
        ) : (
          <p className="inline-feedback">Changes apply immediately and remain on the next launch.</p>
        )}
      </aside>
    </div>
  );
}
