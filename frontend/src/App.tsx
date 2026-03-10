import { useEffect, useState } from "react";
import { EventsOff, EventsOn } from "../wailsjs/runtime/runtime";
import { useFileIntake } from "./features/intake/useFileIntake";
import { usePreferences } from "./features/preferences/usePreferences";
import { AppShell } from "./features/shell/AppShell";
import {
  defaultDiagnostics,
  getDiagnosticsSnapshot,
  type DiagnosticsSnapshot,
} from "./lib/backend";

export default function App() {
  const [showSettings, setShowSettings] = useState(false);
  const [showDetails, setShowDetails] = useState(false);
  const [diagnostics, setDiagnostics] = useState<DiagnosticsSnapshot>(defaultDiagnostics);
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

  useEffect(() => {
    document.documentElement.dataset.theme = preferences.preferences.theme;
  }, [preferences.preferences.theme]);

  useEffect(() => {
    let active = true;

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

    EventsOn("diagnostics:entry", handleSnapshot);

    return () => {
      active = false;
      EventsOff("diagnostics:entry");
    };
  }, []);

  return (
    <AppShell
      diagnostics={diagnostics}
      hasSelection={intake.hasSelection}
      intakeError={intake.error}
      onBrowse={intake.browse}
      onClearIntakeError={intake.clearError}
      onCloseDetails={() => setShowDetails(false)}
      onCloseSettings={() => setShowSettings(false)}
      onOpenDetails={() => setShowDetails(true)}
      onOpenSettings={() => setShowSettings(true)}
      onPreferencesChange={preferences.replacePreferences}
      preferences={preferences.preferences}
      preferencesError={preferences.error}
      selectedFile={intake.selectedFile}
      showDetails={showDetails}
      showSettings={showSettings}
    />
  );
}
