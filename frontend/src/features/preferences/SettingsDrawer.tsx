import type { ModelStatus, Preferences } from "../../lib/backend";

type SettingsDrawerProps = {
  error: string | null;
  modelStatuses: ModelStatus[];
  onClose: () => void;
  onDeleteModel: (modelID: ModelStatus["id"]) => void | Promise<unknown>;
  onDownloadModel: (modelID: ModelStatus["id"]) => void | Promise<unknown>;
  onPreferencesChange: (preferences: Preferences) => void | Promise<unknown>;
  open: boolean;
  preferences: Preferences;
};

export function SettingsDrawer({
  error,
  modelStatuses,
  onClose,
  onDeleteModel,
  onDownloadModel,
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
          <div>
            <p className="section-label">Transcription models</p>
            <h3>Choose the local model that fits this machine.</h3>
          </div>
          <div className="model-library">
            {modelStatuses.map((model) => {
              const isSelected = preferences.model === model.id;
              const canDelete = model.state === "ready";
              const canDownload = model.state === "not_downloaded" || model.state === "failed";

              return (
                <article
                  aria-label={`${model.name} card`}
                  className={`model-card${isSelected ? " model-card-selected" : ""}`}
                  key={model.id}
                >
                  <div className="model-card-header">
                    <div>
                      <p className="section-label">{isSelected ? "Selected model" : "Available model"}</p>
                      <h3>{model.name}</h3>
                    </div>
                    <span
                      aria-label={`${model.name} status`}
                      className={`status-pill status-pill-${model.state}`}
                    >
                      {model.stateLabel}
                    </span>
                  </div>
                  <p className="workspace-copy">{model.description}</p>
                  <div className="model-facts">
                    <span>{model.speedDescription}</span>
                    <span>{model.qualityDescription}</span>
                    <span>{model.systemRequirement}</span>
                  </div>
                  <div className="model-card-actions">
                    <button
                      className={isSelected ? "primary-action" : "ghost-action"}
                      onClick={() =>
                        onPreferencesChange({
                          ...preferences,
                          model: model.id,
                        })
                      }
                      type="button"
                    >
                      {isSelected ? "Selected" : "Use this model"}
                    </button>
                    {canDelete ? (
                      <button className="ghost-action" onClick={() => onDeleteModel(model.id)} type="button">
                        Delete local files
                      </button>
                    ) : (
                      <button
                        className="ghost-action"
                        disabled={!canDownload}
                        onClick={() => onDownloadModel(model.id)}
                        type="button"
                      >
                        {model.state === "downloading" ? "Downloading" : "Download model"}
                      </button>
                    )}
                  </div>
                  {model.error ? (
                    <p className="inline-feedback inline-feedback-error" role="alert">
                      {model.error}
                    </p>
                  ) : null}
                </article>
              );
            })}
          </div>

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
