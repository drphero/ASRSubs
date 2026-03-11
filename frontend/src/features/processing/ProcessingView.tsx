import type { ModelStatus, TranscriptionSnapshot } from "../../lib/backend";

type ProcessingViewProps = {
  selectedModelStatus: ModelStatus | null;
  snapshot: TranscriptionSnapshot;
};

export function ProcessingView({ selectedModelStatus, snapshot }: ProcessingViewProps) {
  return (
    <section className="processing-view" aria-label="processing view">
      <div className="processing-orbit" aria-hidden="true" />
      <div className="processing-card">
        <p className="section-label">Local transcription</p>
        <h2>{snapshot.stage || "Preparing media"}</h2>
        <p className="workspace-copy">
          ASRSubs is running the selected file entirely on this machine. Only the current stage stays visible
          until the run completes or returns you to the workspace.
        </p>
        <div className="processing-meta-grid">
          <div className="processing-meta-card">
            <span>Current file</span>
            <strong>{snapshot.fileName}</strong>
          </div>
          <div className="processing-meta-card">
            <span>Selected model</span>
            <strong>{selectedModelStatus?.name ?? snapshot.modelID}</strong>
          </div>
        </div>
      </div>
    </section>
  );
}
