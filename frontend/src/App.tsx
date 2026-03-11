import { useEffect, useState } from "react";
import { EventsOff, EventsOn } from "../wailsjs/runtime/runtime";
import { useFileIntake } from "./features/intake/useFileIntake";
import { usePreferences } from "./features/preferences/usePreferences";
import { useTranscriptionSession } from "./features/processing/useTranscriptionSession";
import { AppShell } from "./features/shell/AppShell";
import {
  defaultModelSnapshot,
  defaultDiagnostics,
  deleteModel,
  getDiagnosticsSnapshot,
  loadModelSnapshot,
  startModelDownload,
  type DiagnosticsSnapshot,
  type ModelSnapshot,
  type ModelStatus,
} from "./lib/backend";

export default function App() {
  const [showSettings, setShowSettings] = useState(false);
  const [showDetails, setShowDetails] = useState(false);
  const [diagnostics, setDiagnostics] = useState<DiagnosticsSnapshot>(defaultDiagnostics);
  const [modelSnapshot, setModelSnapshot] = useState<ModelSnapshot>(defaultModelSnapshot);
  const [modelError, setModelError] = useState<string | null>(null);
  const preferences = usePreferences();
  const intake = useFileIntake({
    onAcceptedFile(file) {
      void preferences.patchPreferences((current) => ({
        ...current,
        directories: {
          ...current.directories,
          lastOpenDirectory: file.directory,
        },
      }));
    },
  });
  const transcription = useTranscriptionSession({
    selectedFile: intake.selectedFile,
    selectedModel: preferences.preferences.model,
  });

  useEffect(() => {
    document.documentElement.dataset.theme = preferences.preferences.theme;
  }, [preferences.preferences.theme]);

  useEffect(() => {
    let active = true;

    void loadModelSnapshot()
      .then((snapshot) => {
        if (!active) {
          return;
        }

        setModelSnapshot(snapshot);
        setModelError(null);
      })
      .catch(() => {
        if (!active) {
          return;
        }

        setModelError("Model state could not be loaded.");
      });

    void getDiagnosticsSnapshot()
      .then((snapshot) => {
        if (!active) {
          return;
        }

        setDiagnostics(snapshot);
      })
      .catch(() => undefined);

    const handleSnapshot = (snapshot: DiagnosticsSnapshot) => {
      setDiagnostics(snapshot);
    };

    const handleModelSnapshot = (snapshot: ModelSnapshot) => {
      setModelSnapshot(snapshot);
      setModelError(null);
    };

    EventsOn("diagnostics:entry", handleSnapshot);
    EventsOn("models:state", handleModelSnapshot);

    return () => {
      active = false;
      EventsOff("diagnostics:entry");
      EventsOff("models:state");
    };
  }, []);

  const selectedModelStatus =
    modelSnapshot.models.find((model) => model.id === preferences.preferences.model) ?? null;

  async function handleDownloadModel(modelID: ModelStatus["id"]) {
    try {
      const status = await startModelDownload(modelID);
      setModelSnapshot((current) => replaceModelStatus(current, status));
      setModelError(null);
    } catch (caught) {
      setModelError(resolveMessage(caught, "Model download could not be started."));
    }
  }

  async function handleDeleteModel(modelID: ModelStatus["id"]) {
    try {
      const status = await deleteModel(modelID);
      setModelSnapshot((current) => replaceModelStatus(current, status));
      setModelError(null);
    } catch (caught) {
      setModelError(resolveMessage(caught, "Model files could not be deleted."));
    }
  }

  async function handleBrowse() {
    if (!(await transcription.confirmDiscardIfDirty())) {
      return;
    }

    await intake.browse();
  }

  async function handleStartTranscription() {
    if (!(await transcription.confirmDiscardIfDirty())) {
      return;
    }

    await transcription.start();
  }

  return (
    <AppShell
      diagnostics={diagnostics}
      hasSelection={intake.hasSelection}
      intakeError={intake.error}
      modelStatuses={modelSnapshot.models}
      onBrowse={handleBrowse}
      onClearIntakeError={intake.clearError}
      onCloseDetails={() => setShowDetails(false)}
      onCloseSettings={() => setShowSettings(false)}
      onDeleteModel={handleDeleteModel}
      onDownloadModel={handleDownloadModel}
      onOpenDetails={() => setShowDetails(true)}
      onOpenSettings={() => setShowSettings(true)}
      onPreferencesChange={preferences.replacePreferences}
      onRetryTranscription={transcription.retry}
      onSaveSubtitleDraft={transcription.saveEditor}
      onStartTranscription={handleStartTranscription}
      preferences={preferences.preferences}
      preferencesError={preferences.error ?? modelError ?? transcription.error}
      selectedFile={intake.selectedFile}
      selectedModelStatus={selectedModelStatus}
      showDetails={showDetails}
      showSettings={showSettings}
      subtitleEditor={transcription.editor}
      onSubtitleChange={transcription.updateEditorText}
      transcription={transcription.snapshot}
    />
  );
}

function replaceModelStatus(snapshot: ModelSnapshot, next: ModelStatus): ModelSnapshot {
  return {
    ...snapshot,
    models: snapshot.models.map((model) => (model.id === next.id ? next : model)),
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
