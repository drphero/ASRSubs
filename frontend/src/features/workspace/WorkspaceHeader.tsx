import type { MediaMetadata, ModelStatus, RuntimeReadiness } from "../../lib/backend";

type WorkspaceHeaderProps = {
  file: MediaMetadata;
  hasSubtitleDraft: boolean;
  onBrowse: () => void;
  onOpenDetails: () => void;
  onOpenSettings: () => void;
  onStartTranscription: () => void | Promise<unknown>;
  runtimeReadiness: RuntimeReadiness;
  selectedModelStatus: ModelStatus | null;
};

export function WorkspaceHeader({
  file,
  hasSubtitleDraft,
  onBrowse,
  onOpenDetails,
  onOpenSettings,
  onStartTranscription,
  runtimeReadiness,
  selectedModelStatus,
}: WorkspaceHeaderProps) {
  const primaryActionLabel = resolvePrimaryActionLabel(hasSubtitleDraft, runtimeReadiness, selectedModelStatus);
  const modelCopy = resolveModelCopy(runtimeReadiness, selectedModelStatus);

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
          <p className="workspace-model-copy">{modelCopy}</p>
        </div>
        <div className="workspace-action-cluster">
          <div className="workspace-secondary-actions">
            <button className="ghost-action" onClick={onOpenDetails} type="button">
              Details
            </button>
            <button className="ghost-action" onClick={onOpenSettings} type="button">
              Settings
            </button>
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

function resolveModelCopy(runtimeReadiness: RuntimeReadiness, selectedModelStatus: ModelStatus | null) {
  if (runtimeReadiness.state !== "ready") {
    return "Prepare the managed runtime first, then download any missing models from the normal workspace.";
  }
  if (!selectedModelStatus) {
    return "Model state unavailable";
  }
  if (selectedModelStatus.state === "ready") {
    return selectedModelStatus.speedDescription;
  }
  if (selectedModelStatus.state === "downloading") {
    return `${selectedModelStatus.name} is downloading. Transcription will stay blocked until it is ready.`;
  }
  if (selectedModelStatus.state === "failed") {
    return `${selectedModelStatus.name} needs another download attempt before transcription can start.`;
  }
  return `${selectedModelStatus.name} is not downloaded yet. Use the primary action to fetch it without leaving this workspace.`;
}
