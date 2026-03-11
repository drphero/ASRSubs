import type { MediaMetadata, ModelStatus } from "../../lib/backend";

type WorkspaceHeaderProps = {
  file: MediaMetadata;
  onBrowse: () => void;
  onOpenDetails: () => void;
  onOpenSettings: () => void;
  onStartTranscription: () => void | Promise<unknown>;
  selectedModelStatus: ModelStatus | null;
};

export function WorkspaceHeader({
  file,
  onBrowse,
  onOpenDetails,
  onOpenSettings,
  onStartTranscription,
  selectedModelStatus,
}: WorkspaceHeaderProps) {
  return (
    <div className="workspace-header">
      <div>
        <p className="section-label">Selected media</p>
        <h2>{file.name}</h2>
        <p className="workspace-meta">{file.durationLabel}</p>
      </div>
      <div className="workspace-header-actions">
        <div className="workspace-model">
          <span className="model-chip" aria-label="selected model">
            {selectedModelStatus?.name ?? "Model unavailable"}
          </span>
          <span
            aria-label="selected model state"
            className={`status-pill status-pill-${selectedModelStatus?.state ?? "not_downloaded"}`}
          >
            {selectedModelStatus?.stateLabel ?? "Not downloaded"}
          </span>
          <p className="workspace-model-copy">
            {selectedModelStatus?.speedDescription ?? "Model state unavailable"}
          </p>
        </div>
        <button className="ghost-action" onClick={onOpenDetails} type="button">
          Details
        </button>
        <button className="ghost-action" onClick={onOpenSettings} type="button">
          Settings
        </button>
        <button className="primary-action" onClick={onStartTranscription} type="button">
          Start Transcription
        </button>
        <button className="ghost-action" onClick={onBrowse} type="button">
          Replace File
        </button>
      </div>
    </div>
  );
}
