import type { CSSProperties } from "react";
import { useState } from "react";

type MediaDropzoneProps = {
  error: string | null;
  onBrowse: () => void;
  onClearError: () => void;
  selectedModel: string;
};

export function MediaDropzone({
  error,
  onBrowse,
  onClearError,
  selectedModel,
}: MediaDropzoneProps) {
  const [isDragging, setIsDragging] = useState(false);
  const dropTargetStyle = {
    ["--wails-drop-target"]: "drop",
  } as CSSProperties;

  function activateDrag() {
    setIsDragging(true);
    onClearError();
  }

  function deactivateDrag() {
    setIsDragging(false);
  }

  return (
    <section className="landing-panel" aria-label="landing view">
      <div
        aria-label="media dropzone"
        className={`dropzone ${isDragging ? "dropzone-active" : ""}`}
        onDragEnter={activateDrag}
        onDragLeave={deactivateDrag}
        onDragOver={(event) => {
          event.preventDefault();
          activateDrag();
        }}
        onDrop={(event) => {
          event.preventDefault();
          deactivateDrag();
        }}
        style={dropTargetStyle}
      >
        <span className="dropzone-orbit" aria-hidden="true" />
        <div className="dropzone-body">
          <p className="section-label">Drop media</p>
          <h2>Drag a file straight into the center.</h2>
          <div className="dropzone-actions">
            <button className="primary-action" onClick={onBrowse} type="button">
              Browse Media
            </button>
            <span className="secondary-note">Model: {selectedModel}</span>
          </div>
          {error ? (
            <p className="inline-feedback inline-feedback-error" role="alert">
              {error}
            </p>
          ) : null}
        </div>
      </div>
    </section>
  );
}
