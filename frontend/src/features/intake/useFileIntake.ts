import { useEffect, useState } from "react";
import { EventsOff, EventsOn } from "../../../wailsjs/runtime/runtime";
import {
  selectMediaFile,
  validateMediaPath,
  type MediaMetadata,
} from "../../lib/backend";

type UseFileIntakeOptions = {
  onAcceptedFile?: (file: MediaMetadata) => void;
};

export function useFileIntake({ onAcceptedFile }: UseFileIntakeOptions = {}) {
  const [selectedFile, setSelectedFile] = useState<MediaMetadata | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const handleDrop = (_x: number, _y: number, paths: string[]) => {
      const [path] = paths ?? [];
      if (!path) {
        return;
      }

      void acceptPath(path);
    };

    EventsOn("wails:file-drop", handleDrop);

    return () => {
      EventsOff("wails:file-drop");
    };
  }, []);

  async function browse() {
    try {
      const metadata = await selectMediaFile();
      if (!metadata) {
        return;
      }

      setSelectedFile(metadata);
      setError(null);
      onAcceptedFile?.(metadata);
    } catch (caught) {
      setError(resolveMessage(caught, "This file could not be opened."));
    }
  }

  async function acceptPath(path: string) {
    try {
      const metadata = await validateMediaPath(path);
      setSelectedFile(metadata);
      setError(null);
      onAcceptedFile?.(metadata);
    } catch (caught) {
      setError(resolveMessage(caught, "This file could not be read."));
    }
  }

  function clearError() {
    setError(null);
  }

  return {
    browse,
    clearError,
    error,
    hasSelection: selectedFile !== null,
    selectedFile,
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
