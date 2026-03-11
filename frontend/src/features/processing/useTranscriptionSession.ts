import { useEffect, useState } from "react";
import { EventsOff, EventsOn } from "../../../wailsjs/runtime/runtime";
import {
  defaultTranscriptionSnapshot,
  getTranscriptionSnapshot,
  retryTranscription,
  startTranscription,
  type MediaMetadata,
  type ModelID,
  type TranscriptionSnapshot,
} from "../../lib/backend";

type UseTranscriptionSessionOptions = {
  selectedFile: MediaMetadata | null;
  selectedModel: ModelID;
};

export function useTranscriptionSession({ selectedFile, selectedModel }: UseTranscriptionSessionOptions) {
  const [snapshot, setSnapshot] = useState<TranscriptionSnapshot>(defaultTranscriptionSnapshot);
  const [error, setError] = useState<string | null>(null);

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

  async function start() {
    if (!selectedFile) {
      setError("Choose a media file before starting.");
      return;
    }

    try {
      const next = await startTranscription({
        mediaPath: selectedFile.path,
        modelID: selectedModel,
      });
      setSnapshot(next);
      setError(null);
    } catch (caught) {
      setError(resolveMessage(caught, "Transcription could not be started."));
    }
  }

  async function retry() {
    try {
      const next = await retryTranscription();
      setSnapshot(next);
      setError(null);
    } catch (caught) {
      setError(resolveMessage(caught, "Transcription could not be retried."));
    }
  }

  return {
    error,
    retry,
    snapshot,
    start,
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
