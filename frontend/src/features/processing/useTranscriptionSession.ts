import { useEffect, useRef, useState } from "react";
import { EventsOff, EventsOn } from "../../../wailsjs/runtime/runtime";
import {
  confirmDiscardSubtitleDraft,
  defaultTranscriptionSnapshot,
  getSubtitleDraft,
  getTranscriptionSnapshot,
  retryTranscription,
  saveSubtitleDraft,
  startTranscription,
  type MediaMetadata,
  type ModelID,
  type SubtitleDraft,
  type TranscriptionSnapshot,
} from "../../lib/backend";

type UseTranscriptionSessionOptions = {
  selectedFile: MediaMetadata | null;
  selectedModel: ModelID;
};

type SubtitleEditorInternalState = {
  baselineText: string;
  draft: SubtitleDraft | null;
  dirty: boolean;
  focusRequestId: number;
  isSaving: boolean;
  saveFeedback: SubtitleEditorSaveFeedback | null;
  text: string;
};

export type SubtitleEditorSaveFeedback = {
  kind: "saved" | "canceled" | "invalid";
  message: string;
};

export type SubtitleEditorSession = Omit<SubtitleEditorInternalState, "baselineText">;

const emptyEditorState: SubtitleEditorInternalState = {
  baselineText: "",
  draft: null,
  dirty: false,
  focusRequestId: 0,
  isSaving: false,
  saveFeedback: null,
  text: "",
};

export function useTranscriptionSession({ selectedFile, selectedModel }: UseTranscriptionSessionOptions) {
  const [snapshot, setSnapshot] = useState<TranscriptionSnapshot>(defaultTranscriptionSnapshot);
  const [editor, setEditor] = useState<SubtitleEditorInternalState>(emptyEditorState);
  const [error, setError] = useState<string | null>(null);
  const previousSnapshotRef = useRef(defaultTranscriptionSnapshot);
  const editorRef = useRef(emptyEditorState);

  useEffect(() => {
    editorRef.current = editor;
  }, [editor]);

  useEffect(() => {
    let active = true;

    void getTranscriptionSnapshot()
      .then((current) => {
        if (!active) {
          return;
        }

        setSnapshot(current);
        setError(null);
      })
      .catch(() => {
        if (!active) {
          return;
        }

        setError("Transcription state could not be loaded.");
      });

    const handleSnapshot = (next: TranscriptionSnapshot) => {
      setSnapshot(next);
      setError(null);
    };

    EventsOn("transcription:state", handleSnapshot);

    return () => {
      active = false;
      EventsOff("transcription:state");
    };
  }, []);

  useEffect(() => {
    const previous = previousSnapshotRef.current;
    previousSnapshotRef.current = snapshot;

    const completedSuccessfully =
      previous.active && !snapshot.active && snapshot.failureSummary === "" && snapshot.filePath !== "";

    if (!completedSuccessfully) {
      return;
    }

    let cancelled = false;
    setEditor((current) => ({
      ...current,
      isSaving: false,
      saveFeedback: null,
    }));

    void getSubtitleDraft()
      .then((draft) => {
        if (cancelled) {
          return;
        }

        setEditor((current) => ({
          baselineText: draft.text,
          draft,
          dirty: false,
          focusRequestId: current.focusRequestId + 1,
          isSaving: false,
          saveFeedback: null,
          text: draft.text,
        }));
        setError(null);
      })
      .catch(() => {
        if (cancelled) {
          return;
        }

        setError("Subtitle draft could not be loaded.");
      });

    return () => {
      cancelled = true;
    };
  }, [snapshot]);

  useEffect(() => {
    if (!selectedFile || !editorRef.current.draft || snapshot.active) {
      return;
    }

    if (selectedFile.path !== editorRef.current.draft.sourceFilePath) {
      setEditor(emptyEditorState);
    }
  }, [selectedFile, snapshot.active]);

  async function start() {
    if (!selectedFile) {
      setError("Choose a media file before starting.");
      return false;
    }

    try {
      const next = await startTranscription({
        mediaPath: selectedFile.path,
        modelID: selectedModel,
      });
      setSnapshot(next);
      setEditor(emptyEditorState);
      setError(null);
      return true;
    } catch (caught) {
      setError(resolveMessage(caught, "Transcription could not be started."));
      return false;
    }
  }

  async function retry() {
    try {
      const next = await retryTranscription();
      setSnapshot(next);
      setError(null);
      return true;
    } catch (caught) {
      setError(resolveMessage(caught, "Transcription could not be retried."));
      return false;
    }
  }

  function updateEditorText(text: string) {
    setEditor((current) => ({
      ...current,
      dirty: text !== current.baselineText,
      saveFeedback: null,
      text,
    }));
  }

  async function saveEditor() {
    const current = editorRef.current;
    if (!current.draft) {
      return;
    }

    setEditor((state) => ({
      ...state,
      isSaving: true,
    }));

    try {
      const result = await saveSubtitleDraft({
        suggestedFilename: current.draft.suggestedFilename,
        text: current.text,
      });

      if (result.status === "saved") {
        setEditor((state) => ({
          ...state,
          baselineText: state.text,
          dirty: false,
          isSaving: false,
          saveFeedback: {
            kind: "saved",
            message: result.fileName ? `Saved ${result.fileName}.` : "Subtitle file saved.",
          },
        }));
        return;
      }

      if (result.status === "canceled") {
        setEditor((state) => ({
          ...state,
          isSaving: false,
          saveFeedback: {
            kind: "canceled",
            message: "Save canceled.",
          },
        }));
        return;
      }

      setEditor((state) => ({
        ...state,
        isSaving: false,
        saveFeedback: {
          kind: "invalid",
          message: result.validationIssue
            ? `Line ${result.validationIssue.line}: ${result.validationIssue.message}`
            : "The subtitle file is malformed.",
        },
      }));
    } catch (caught) {
      setError(resolveMessage(caught, "Subtitle file could not be saved."));
      setEditor((state) => ({
        ...state,
        isSaving: false,
      }));
    }
  }

  async function confirmDiscardIfDirty() {
    if (!editorRef.current.draft || !editorRef.current.dirty) {
      return true;
    }

    try {
      return await confirmDiscardSubtitleDraft();
    } catch (caught) {
      setError(resolveMessage(caught, "Discard confirmation could not be opened."));
      return false;
    }
  }

  return {
    confirmDiscardIfDirty,
    editor,
    error,
    retry,
    saveEditor,
    snapshot,
    start,
    updateEditorText,
  };
}

function resolveMessage(caught: unknown, fallback: string) {
  if (caught instanceof Error && caught.message.trim().length > 0) {
    return caught.message;
  }

  if (typeof caught === "string" && caught.trim().length > 0) {
    return caught;
  }

  return fallback;
}
