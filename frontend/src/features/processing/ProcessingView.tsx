import type { ModelStatus, TranscriptionSnapshot } from "../../lib/backend";

type ProcessingViewProps = {
  selectedModelStatus: ModelStatus | null;
  snapshot: TranscriptionSnapshot;
};

export function ProcessingView({ selectedModelStatus, snapshot }: ProcessingViewProps) {
  const partLabel =
    snapshot.partCount > 1 && snapshot.partIndex > 0 ? `Part ${snapshot.partIndex} of ${snapshot.partCount}` : null;
  const downloadLabel =
    snapshot.stage === "Downloading model" && snapshot.downloadTargetName ? snapshot.downloadTargetName : null;

  return (
    <section className="processing-view" aria-label="processing view">
      <div className="processing-orbit" aria-hidden="true" />
      <div className="processing-card">
        <p className="section-label">Local transcription</p>
        <h2>{snapshot.stage || "Preparing media"}</h2>
        {partLabel ? <p className="inline-feedback">{partLabel}</p> : null}
        {downloadLabel ? <p className="inline-feedback">{downloadLabel}</p> : null}
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
