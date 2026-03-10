import type { DiagnosticsSnapshot, MediaMetadata, Preferences } from "../../lib/backend";
import { DetailsPanel } from "../diagnostics/DetailsPanel";
import { MediaDropzone } from "../intake/MediaDropzone";
import { SettingsDrawer } from "../preferences/SettingsDrawer";
import { SelectedFileSummary } from "../workspace/SelectedFileSummary";
import { WorkspaceHeader } from "../workspace/WorkspaceHeader";

type AppShellProps = {
  diagnostics: DiagnosticsSnapshot;
  hasSelection: boolean;
  intakeError: string | null;
  onBrowse: () => void;
  onClearIntakeError: () => void;
  onCloseDetails: () => void;
  onCloseSettings: () => void;
  onOpenDetails: () => void;
  onOpenSettings: () => void;
  onPreferencesChange: (preferences: Preferences) => void | Promise<unknown>;
  preferences: Preferences;
  preferencesError: string | null;
  selectedFile: MediaMetadata | null;
  showDetails: boolean;
  showSettings: boolean;
};

export function AppShell({
  diagnostics,
  hasSelection,
  intakeError,
  onBrowse,
  onClearIntakeError,
  onCloseDetails,
  onCloseSettings,
  onOpenDetails,
  onOpenSettings,
  onPreferencesChange,
  preferences,
  preferencesError,
  selectedFile,
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
                  selectedModel={preferences.model}
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
        onClose={onCloseSettings}
        onPreferencesChange={onPreferencesChange}
        open={showSettings}
        preferences={preferences}
      />
      <DetailsPanel onClose={onCloseDetails} open={showDetails} snapshot={diagnostics} />
    </div>
  );
}
