import { useEffect, useState } from "react";
import { EventsOff, EventsOn } from "../wailsjs/runtime/runtime";
import { useFileIntake } from "./features/intake/useFileIntake";
import { usePreferences } from "./features/preferences/usePreferences";
import { useTranscriptionSession } from "./features/processing/useTranscriptionSession";
import { AppShell } from "./features/shell/AppShell";
import {
  defaultModelSnapshot,
  defaultDiagnostics,
  defaultRuntimeReadiness,
  deleteModel,
  ensureRuntimeReady,
  getDiagnosticsSnapshot,
  getRuntimeReadiness,
  loadModelSnapshot,
  startModelDownload,
  type DiagnosticsSnapshot,
  type ModelSnapshot,
  type ModelStatus,
  type RuntimeReadiness,
} from "./lib/backend";

export default function App() {
  const [showSettings, setShowSettings] = useState(false);
  const [showDetails, setShowDetails] = useState(false);
  const [showRuntimeOverlay, setShowRuntimeOverlay] = useState(false);
  const [runtimePreparing, setRuntimePreparing] = useState(false);
  const [diagnostics, setDiagnostics] = useState<DiagnosticsSnapshot>(defaultDiagnostics);
  const [modelSnapshot, setModelSnapshot] = useState<ModelSnapshot>(defaultModelSnapshot);
  const [modelError, setModelError] = useState<string | null>(null);
  const [runtimeReadiness, setRuntimeReadiness] = useState<RuntimeReadiness>(defaultRuntimeReadiness);
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

    void getRuntimeReadiness()
      .then((readiness) => {
        if (!active) {
          return;
        }

        setRuntimeReadiness(readiness);
        setShowRuntimeOverlay(readiness.state !== "ready");
      })
      .catch(() => {
        if (!active) {
          return;
        }

        setRuntimeReadiness({
          ...defaultRuntimeReadiness,
          state: "failed",
          detail: "Managed runtime state could not be loaded.",
        });
        setShowRuntimeOverlay(true);
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
    if (runtimeReadiness.state !== "ready") {
      setShowRuntimeOverlay(true);
      return;
    }

    if (selectedModelStatus?.state !== "ready") {
      await handleModelAction(selectedModelStatus);
      return;
    }

    if (!(await transcription.confirmDiscardIfDirty())) {
      return;
    }

    await transcription.start();
  }

  async function handlePrepareRuntime() {
    setRuntimePreparing(true);
    try {
      const readiness = await ensureRuntimeReady();
      setRuntimeReadiness(readiness);
      setShowRuntimeOverlay(readiness.state !== "ready");
      setModelError(null);
    } catch (caught) {
      setRuntimeReadiness((current) => ({
        ...current,
        state: "failed",
        detail: resolveMessage(caught, "Managed runtime could not be prepared."),
      }));
      setShowRuntimeOverlay(true);
    } finally {
      setRuntimePreparing(false);
    }
  }

  async function handleModelAction(status: ModelStatus | null) {
    if (!status) {
      setModelError("Selected model state is unavailable.");
      return;
    }

    if (status.state === "downloading") {
      setModelError(`${status.name} is still downloading.`);
      return;
    }

    setModelError(`${status.name} must be downloaded before transcription can start.`);
    await handleDownloadModel(status.id);
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
      onCloseRuntimeSetup={() => setShowRuntimeOverlay(false)}
      onPreferencesChange={preferences.replacePreferences}
      onPrepareRuntime={handlePrepareRuntime}
      onRetryTranscription={transcription.retry}
      onSaveSubtitleDraft={transcription.saveEditor}
      onStartTranscription={handleStartTranscription}
      preferences={preferences.preferences}
      preferencesError={preferences.error ?? modelError ?? transcription.error}
      runtimePreparing={runtimePreparing}
      runtimeReadiness={runtimeReadiness}
      selectedFile={intake.selectedFile}
      selectedModelStatus={selectedModelStatus}
      showDetails={showDetails}
      showRuntimeSetup={showRuntimeOverlay}
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
