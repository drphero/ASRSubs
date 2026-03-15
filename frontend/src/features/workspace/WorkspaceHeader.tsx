import type { MediaMetadata, ModelStatus, RuntimeReadiness } from "../../lib/backend";

type WorkspaceHeaderProps = {
  file: MediaMetadata;
  hasSubtitleDraft: boolean;
  onBrowse: () => void;
  onStartTranscription: () => void | Promise<unknown>;
  runtimeReadiness: RuntimeReadiness;
  selectedModelStatus: ModelStatus | null;
};

export function WorkspaceHeader({
  file,
  hasSubtitleDraft,
  onBrowse,
  onStartTranscription,
  runtimeReadiness,
  selectedModelStatus,
}: WorkspaceHeaderProps) {
  const primaryActionLabel = resolvePrimaryActionLabel(hasSubtitleDraft, runtimeReadiness, selectedModelStatus);

  return (
    <div className={`workspace-header ${hasSubtitleDraft ? "workspace-header-complete" : ""}`.trim()}>
      <div className="workspace-header-copy">
        <p className="section-label">{hasSubtitleDraft ? "Subtitle workspace" : "Selected media"}</p>
        <h2>{file.name}</h2>
        <div className="workspace-header-meta">
          <span className="workspace-meta-pill">{file.durationLabel}</span>
          <span className="workspace-meta-pill">{file.extension.replace(".", "").toUpperCase()}</span>
          <span className="workspace-meta-pill">{file.directory.split("/").filter(Boolean).pop() || file.directory}</span>
          {hasSubtitleDraft ? <span className="workspace-meta-pill">Draft ready</span> : null}
        </div>
      </div>
      <div className="workspace-header-side">
        <div className="workspace-status-row">
          <span className="model-chip" aria-label="selected model">
            {selectedModelStatus?.name ?? "Model unavailable"}
          </span>
          <span
            aria-label="selected model state"
            className={`status-pill status-pill-${selectedModelStatus?.state ?? "not_downloaded"}`}
          >
            {selectedModelStatus?.stateLabel ?? "Not downloaded"}
          </span>
        </div>
        <div className="workspace-primary-actions">
          <button className="primary-action" onClick={onStartTranscription} type="button">
            {primaryActionLabel}
          </button>
          <button className="ghost-action" onClick={onBrowse} type="button">
            Replace File
          </button>
        </div>
      </div>
    </div>
  );
}

function resolvePrimaryActionLabel(
  hasSubtitleDraft: boolean,
  runtimeReadiness: RuntimeReadiness,
  selectedModelStatus: ModelStatus | null,
) {
  if (runtimeReadiness.state !== "ready") {
    return "Prepare Runtime";
  }
  if (!selectedModelStatus || selectedModelStatus.state === "not_downloaded") {
    return "Download Model";
  }
  if (selectedModelStatus.state === "downloading") {
    return "Model Downloading...";
  }
  if (selectedModelStatus.state === "failed") {
    return "Retry Model Download";
  }
  return hasSubtitleDraft ? "Run Again" : "Start Transcription";
}
