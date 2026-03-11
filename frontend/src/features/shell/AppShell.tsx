import type { DiagnosticsSnapshot, MediaMetadata, ModelStatus, Preferences } from "../../lib/backend";
import { DetailsPanel } from "../diagnostics/DetailsPanel";
import { MediaDropzone } from "../intake/MediaDropzone";
import { SettingsDrawer } from "../preferences/SettingsDrawer";
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
  onPreferencesChange: (preferences: Preferences) => void | Promise<unknown>;
  preferences: Preferences;
  preferencesError: string | null;
  selectedFile: MediaMetadata | null;
  selectedModelStatus: ModelStatus | null;
  showDetails: boolean;
  showSettings: boolean;
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
  onPreferencesChange,
  preferences,
  preferencesError,
  selectedFile,
  selectedModelStatus,
  showDetails,
  showSettings,
}: AppShellProps) {
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
        {hasSelection ? (
          <section className="workspace-view" aria-label="workspace view">
            {selectedFile ? (
              <>
                <WorkspaceHeader
                  file={selectedFile}
                  onBrowse={onBrowse}
                  onOpenDetails={onOpenDetails}
                  onOpenSettings={onOpenSettings}
                  selectedModelStatus={selectedModelStatus}
                />
                <SelectedFileSummary error={intakeError} file={selectedFile} />
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
        open={showSettings}
        preferences={preferences}
      />
      <DetailsPanel onClose={onCloseDetails} open={showDetails} snapshot={diagnostics} />
    </div>
  );
}
