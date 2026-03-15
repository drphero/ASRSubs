import type { RuntimeReadiness } from "../../lib/backend";

type RuntimeSetupOverlayProps = {
  busy: boolean;
  onClose: () => void;
  onOpenDetails: () => void;
  onPrepare: () => void | Promise<unknown>;
  open: boolean;
  readiness: RuntimeReadiness;
};

export function RuntimeSetupOverlay({
  busy,
  onClose,
  onOpenDetails,
  onPrepare,
  open,
  readiness,
}: RuntimeSetupOverlayProps) {
  if (!open) {
    return null;
  }

  const failed = readiness.state === "failed";
  const title = failed ? "Runtime setup needs attention." : "Prepare the local runtime before the first run.";
  const summary = failed
    ? "Runtime setup did not finish. Retry it here or inspect diagnostics."
    : "ASRSubs needs to prepare its local runtime before the first transcription.";
  const actionLabel = busy ? "Preparing runtime..." : failed ? "Retry Runtime Setup" : "Prepare Runtime";

  return (
    <div className="overlay-shell runtime-overlay-shell" role="presentation">
      <button aria-label="Dismiss runtime setup" className="overlay-backdrop" onClick={onClose} type="button" />

      <section aria-label="runtime setup overlay" className="runtime-overlay-panel">
        <div className="runtime-overlay-copy">
          <p className="section-label">Managed runtime</p>
          <h2>{title}</h2>
          <p className="workspace-copy">{summary}</p>
        </div>

        <div className="runtime-overlay-grid">
          <div className="summary-stat">
            <span>Status</span>
            <strong>{formatState(readiness.state)}</strong>
          </div>
          <div className="summary-stat">
            <span>Install location</span>
            <strong>{readiness.rootDir || "App config directory"}</strong>
          </div>
          <div className="summary-stat">
            <span>Python path</span>
            <strong>{readiness.pythonPath || "Prepared during setup"}</strong>
          </div>
          <div className="summary-stat">
            <span>Worker script</span>
            <strong>{readiness.workerPath || "Bundled with the app"}</strong>
          </div>
        </div>

        <div className="runtime-overlay-message">
          <span className={`status-pill status-pill-${failed ? "failed" : "not_downloaded"}`}>
            {failed ? "Retry available" : "Confirmation required"}
          </span>
          <p>{readiness.detail || "The normal workspace stays available while setup runs."}</p>
        </div>

        <div className="runtime-overlay-actions">
          <button className="primary-action" disabled={busy} onClick={onPrepare} type="button">
            {actionLabel}
          </button>
          <button className="ghost-action" onClick={onOpenDetails} type="button">
            Details
          </button>
          <button className="ghost-action" onClick={onClose} type="button">
            Not Now
          </button>
        </div>
      </section>
    </div>
  );
}

function formatState(state: string) {
  if (state === "ready") {
    return "Ready";
  }
  if (state === "failed") {
    return "Needs attention";
  }
  if (state === "missing") {
    return "Not prepared";
  }
  return state;
}
