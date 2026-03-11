import type { MediaMetadata } from "../../lib/backend";

type SelectedFileSummaryProps = {
  error: string | null;
  file: MediaMetadata;
  onRetryTranscription: () => void | Promise<unknown>;
  transcriptionFailure: string;
  transcriptionRetryAvailable: boolean;
};

export function SelectedFileSummary({
  error,
  file,
  onRetryTranscription,
  transcriptionFailure,
  transcriptionRetryAvailable,
}: SelectedFileSummaryProps) {
  return (
    <section className="workspace-card selected-file-card" aria-label="selected file summary">
      <div className="summary-grid">
        <div>
          <p className="section-label">File</p>
          <h3>{file.name}</h3>
        </div>
        <div>
          <p className="section-label">Duration</p>
          <h3>{file.durationLabel}</h3>
        </div>
      </div>
      <p className="workspace-copy">
        The subtitle editor and save flow arrive in the next plans. This phase keeps the
        chosen file, settings, and app activity visible in one place.
      </p>
      {transcriptionFailure ? (
        <div className="transcription-inline-state">
          <p className="inline-feedback inline-feedback-error" role="alert">
            {transcriptionFailure}
          </p>
          {transcriptionRetryAvailable ? (
            <button className="ghost-action" onClick={onRetryTranscription} type="button">
              Retry transcription
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
