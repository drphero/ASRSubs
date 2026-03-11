import type { MediaMetadata } from "../../lib/backend";

type SelectedFileSummaryProps = {
  error: string | null;
  file: MediaMetadata;
  onRetryTranscription: () => void | Promise<unknown>;
  transcriptionFailedStage: string;
  transcriptionFailure: string;
  transcriptionRetryAvailable: boolean;
};

export function SelectedFileSummary({
  error,
  file,
  onRetryTranscription,
  transcriptionFailedStage,
  transcriptionFailure,
  transcriptionRetryAvailable,
}: SelectedFileSummaryProps) {
  const retryLabel = transcriptionFailedStage ? `Retry ${transcriptionFailedStage.toLowerCase()}` : "Retry transcription";

  return (
    <section className="workspace-card selected-file-card" aria-label="selected file summary">
      <div className="selected-file-intro">
        <div>
          <p className="section-label">Ready to run</p>
          <h3>Local transcription is set up for this file.</h3>
        </div>
        <p className="workspace-copy">
          The subtitle editor arrives in the next phase. For now, this workspace keeps the file facts,
          model state, diagnostics, and retry flow in one place without repeating the same headline twice.
        </p>
      </div>
      <div className="summary-grid">
        <div className="summary-stat">
          <span>Source folder</span>
          <strong>{file.directory}</strong>
        </div>
        <div className="summary-stat">
          <span>Duration</span>
          <strong>{file.durationLabel}</strong>
        </div>
        <div className="summary-stat">
          <span>Format</span>
          <strong>{file.extension.replace(".", "").toUpperCase()}</strong>
        </div>
        <div className="summary-stat">
          <span>File size</span>
          <strong>{formatFileSize(file.sizeBytes)}</strong>
        </div>
      </div>
      {transcriptionFailure ? (
        <div className="transcription-inline-state">
          <p className="inline-feedback inline-feedback-error" role="alert">
            {transcriptionFailedStage ? `${transcriptionFailedStage} failed. ${transcriptionFailure}` : transcriptionFailure}
          </p>
          {transcriptionRetryAvailable ? (
            <button className="ghost-action" onClick={onRetryTranscription} type="button">
              {retryLabel}
            </button>
          ) : null}
        </div>
      ) : null}
      {error ? (
        <p className="inline-feedback inline-feedback-error" role="alert">
          {error}
        </p>
      ) : null}
    </section>
  );
}

function formatFileSize(sizeBytes: number) {
  if (sizeBytes < 1024) {
    return `${sizeBytes} B`;
  }

  const units = ["KB", "MB", "GB"];
  let value = sizeBytes / 1024;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unitIndex]}`;
}
