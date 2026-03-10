import type { MediaMetadata } from "../../lib/backend";

type WorkspaceHeaderProps = {
  file: MediaMetadata;
  onBrowse: () => void;
  onOpenDetails: () => void;
  onOpenSettings: () => void;
  selectedModel: string;
};

export function WorkspaceHeader({
  file,
  onBrowse,
  onOpenDetails,
  onOpenSettings,
  selectedModel,
}: WorkspaceHeaderProps) {
  return (
    <div className="workspace-header">
      <div>
        <p className="section-label">Selected media</p>
        <h2>{file.name}</h2>
        <p className="workspace-meta">{file.durationLabel}</p>
      </div>
      <div className="workspace-header-actions">
        <span className="model-chip" aria-label="selected model">
          {selectedModel}
        </span>
        <button className="ghost-action" onClick={onOpenDetails} type="button">
          Details
        </button>
        <button className="ghost-action" onClick={onOpenSettings} type="button">
          Settings
        </button>
        <button className="primary-action" onClick={onBrowse} type="button">
          Replace File
        </button>
      </div>
    </div>
  );
}
