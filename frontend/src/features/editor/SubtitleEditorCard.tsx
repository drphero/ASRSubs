import { useEffect, useRef } from "react";
import type { SubtitleDraft } from "../../lib/backend";
import type { SubtitleEditorSaveFeedback } from "../processing/useTranscriptionSession";

type SubtitleEditorCardProps = {
  draft: SubtitleDraft;
  text: string;
  dirty: boolean;
  focusRequestId: number;
  isSaving?: boolean;
  saveFeedback?: SubtitleEditorSaveFeedback | null;
  onChange: (text: string) => void;
  onSave?: () => void | Promise<unknown>;
};

export function SubtitleEditorCard({
  draft,
  text,
  dirty,
  focusRequestId,
  isSaving = false,
  saveFeedback = null,
  onChange,
  onSave,
}: SubtitleEditorCardProps) {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const appliedFocusRef = useRef(0);
  const statusLabel = dirty ? "Unsaved edits" : saveFeedback?.kind === "saved" ? "Saved" : "Draft ready";

  useEffect(() => {
    if (!textareaRef.current || focusRequestId === 0 || appliedFocusRef.current === focusRequestId) {
      return;
    }

    textareaRef.current.focus();
    textareaRef.current.setSelectionRange(0, 0);
    appliedFocusRef.current = focusRequestId;
  }, [focusRequestId]);

  return (
    <section className="workspace-card editor-card" aria-label="subtitle editor">
      <div className="editor-card-header">
        <div>
          <p className="section-label">Completed subtitle draft</p>
          <h3>{draft.suggestedFilename}</h3>
        </div>
        <div className="editor-card-status">
          <span className={`status-pill ${dirty ? "editor-status-dirty" : "editor-status-saved"}`}>
            {statusLabel}
          </span>
          {onSave ? (
            <button className="primary-action" disabled={isSaving} onClick={onSave} type="button">
              {isSaving ? "Saving..." : "Save As .srt"}
            </button>
          ) : null}
        </div>
      </div>

      <p className="workspace-copy editor-copy">
        The raw subtitle file stays local in this workspace. The caret lands at the top so you can start correcting the
        generated `.srt` immediately.
      </p>

      <label className="field editor-field">
        <span>Editable SRT text</span>
        <textarea
          aria-label="Editable subtitle text"
          className="editor-textarea"
          onChange={(event) => onChange(event.currentTarget.value)}
          ref={textareaRef}
          spellCheck={false}
          value={text}
        />
      </label>

      <div className="editor-meta-row">
        <span className="workspace-meta-pill">{draft.sourceFileName}</span>
        <span className="workspace-meta-pill">Raw .srt</span>
      </div>

      {saveFeedback ? (
        <p
          className={`inline-feedback ${
            saveFeedback.kind === "invalid" ? "inline-feedback-error" : "inline-feedback-success"
          }`}
          role={saveFeedback.kind === "invalid" ? "alert" : "status"}
        >
          {saveFeedback.message}
        </p>
      ) : null}
    </section>
  );
}
