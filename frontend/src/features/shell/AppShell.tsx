import type {
  DiagnosticsSnapshot,
  MediaMetadata,
  ModelStatus,
  Preferences,
  RuntimeReadiness,
} from "../../lib/backend";
import { SubtitleEditorCard } from "../editor/SubtitleEditorCard";
import type { SubtitleEditorSession } from "../processing/useTranscriptionSession";
import { DetailsPanel } from "../diagnostics/DetailsPanel";
import { MediaDropzone } from "../intake/MediaDropzone";
import { SettingsDrawer } from "../preferences/SettingsDrawer";
import { ProcessingView } from "../processing/ProcessingView";
import { RuntimeSetupOverlay } from "../runtime/RuntimeSetupOverlay";
import { SelectedFileSummary } from "../workspace/SelectedFileSummary";
import { WorkspaceHeader } from "../workspace/WorkspaceHeader";

type AppShellProps = {
  diagnostics: DiagnosticsSnapshot;
  hasSelection: boolean;
  intakeError: string | null;
  modelStatuses: ModelStatus[];
  onBrowse: () => void;
  onClearIntakeError: () => void;
  onCloseDetails: () => void;
  onCloseSettings: () => void;
  onDeleteModel: (modelID: ModelStatus["id"]) => void | Promise<unknown>;
  onDownloadModel: (modelID: ModelStatus["id"]) => void | Promise<unknown>;
  onOpenDetails: () => void;
  onOpenSettings: () => void;
  onCloseRuntimeSetup: () => void;
  onPreferencesChange: (preferences: Preferences) => void | Promise<unknown>;
  onPrepareRuntime: () => void | Promise<unknown>;
  onRetryTranscription: () => void | Promise<unknown>;
  onSaveSubtitleDraft: () => void | Promise<unknown>;
  onStartTranscription: () => void | Promise<unknown>;
  onSubtitleChange: (text: string) => void;
  preferences: Preferences;
  preferencesError: string | null;
  runtimePreparing: boolean;
  runtimeReadiness: RuntimeReadiness;
  selectedFile: MediaMetadata | null;
  selectedModelStatus: ModelStatus | null;
  showDetails: boolean;
  showRuntimeSetup: boolean;
  showSettings: boolean;
  subtitleEditor: SubtitleEditorSession;
  transcription: import("../../lib/backend").TranscriptionSnapshot;
};

export function AppShell({
  diagnostics,
  hasSelection,
  intakeError,
  modelStatuses,
  onBrowse,
  onClearIntakeError,
  onCloseDetails,
  onCloseSettings,
  onDeleteModel,
  onDownloadModel,
  onOpenDetails,
  onOpenSettings,
  onCloseRuntimeSetup,
  onPreferencesChange,
  onPrepareRuntime,
  onRetryTranscription,
  onSaveSubtitleDraft,
  onStartTranscription,
  onSubtitleChange,
  preferences,
  preferencesError,
  runtimePreparing,
  runtimeReadiness,
  selectedFile,
  selectedModelStatus,
  showDetails,
  showRuntimeSetup,
  showSettings,
  subtitleEditor,
  transcription,
}: AppShellProps) {
  const hasSubtitleDraft = subtitleEditor.draft !== null;

  return (
    <div className="app-shell">
      <div className="ambient ambient-left" aria-hidden="true" />
      <div className="ambient ambient-right" aria-hidden="true" />

      <header className="topbar">
        <div>
          <p className="eyebrow">ASRSubs</p>
          <h1>Local subtitles with zero ceremony.</h1>
        </div>
        <div className="topbar-actions">
          <button className="ghost-action" onClick={onOpenDetails} type="button">
            Details
          </button>
          <button className="ghost-action" onClick={onOpenSettings} type="button">
            Settings
          </button>
        </div>
      </header>

      <main className="shell-main">
        {transcription.active ? (
          <ProcessingView selectedModelStatus={selectedModelStatus} snapshot={transcription} />
        ) : hasSelection ? (
          <section
            aria-label="workspace view"
            className={`workspace-view ${hasSubtitleDraft ? "workspace-view-complete" : ""}`.trim()}
          >
            {selectedFile ? (
              <>
                {hasSubtitleDraft ? (
                  <section className="workspace-card workspace-success-card" aria-label="completed editing state">
                    <p className="section-label">Transcription complete</p>
                    <h2>Subtitle draft ready for final edits.</h2>
                  </section>
                ) : null}
                <WorkspaceHeader
                  hasSubtitleDraft={hasSubtitleDraft}
                  file={selectedFile}
                  onBrowse={onBrowse}
                  onStartTranscription={onStartTranscription}
                  runtimeReadiness={runtimeReadiness}
                  selectedModelStatus={selectedModelStatus}
                />
                {subtitleEditor.draft ? (
                  <SubtitleEditorCard
                    dirty={subtitleEditor.dirty}
                    draft={subtitleEditor.draft}
                    focusRequestId={subtitleEditor.focusRequestId}
                    isSaving={subtitleEditor.isSaving}
                    onChange={onSubtitleChange}
                    onSave={onSaveSubtitleDraft}
                    saveFeedback={subtitleEditor.saveFeedback}
                    text={subtitleEditor.text}
                  />
                ) : (
                  <SelectedFileSummary
                    error={intakeError}
                    file={selectedFile}
                    onRetryTranscription={onRetryTranscription}
                    transcriptionFailedStage={transcription.failedStage}
                    transcriptionFailure={transcription.failureSummary}
                    transcriptionRetryAvailable={transcription.canRetry}
                  />
                )}
              </>
            ) : null}
          </section>
        ) : (
          <MediaDropzone
            error={intakeError}
            onBrowse={onBrowse}
            onClearError={onClearIntakeError}
            selectedModel={preferences.model}
          />
        )}
      </main>

      <SettingsDrawer
        error={preferencesError}
        modelStatuses={modelStatuses}
        onClose={onCloseSettings}
        onDeleteModel={onDeleteModel}
        onDownloadModel={onDownloadModel}
        onPreferencesChange={onPreferencesChange}
        onPrepareRuntime={onPrepareRuntime}
        open={showSettings}
        preferences={preferences}
        runtimePreparing={runtimePreparing}
        runtimeReadiness={runtimeReadiness}
      />
      <DetailsPanel onClose={onCloseDetails} open={showDetails} snapshot={diagnostics} />
      <RuntimeSetupOverlay
        busy={runtimePreparing}
        onClose={onCloseRuntimeSetup}
        onOpenDetails={onOpenDetails}
        onPrepare={onPrepareRuntime}
        open={showRuntimeSetup}
        readiness={runtimeReadiness}
      />
    </div>
  );
}
