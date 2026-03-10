export type ThemeMode = "dark" | "light";
export type ModelID = "Qwen3-ASR-0.6B" | "Qwen3-ASR-1.7B";

export interface MediaMetadata {
  path: string;
  name: string;
  extension: string;
  directory: string;
  sizeBytes: number;
  durationSeconds: number;
  durationLabel: string;
  hasKnownDuration: boolean;
}

export interface OutputPreferences {
  maxLineLength: number;
  linesPerSubtitle: number;
}

export interface DirectoryPreferences {
  lastOpenDirectory: string;
  lastSaveDirectory: string;
}

export interface ProcessingPreferences {
  alignmentChunkMinutes: number;
}

export interface Preferences {
  version: number;
  model: ModelID;
  theme: ThemeMode;
  output: OutputPreferences;
  directories: DirectoryPreferences;
  processing: ProcessingPreferences;
}

export interface DiagnosticsEntry {
  id: string;
  level: "info" | "warning" | "error";
  source: string;
  message: string;
  timestamp: string;
}

export interface DiagnosticsSummary {
  title: string;
  message: string;
  level: "info" | "warning" | "error";
}

export interface DiagnosticsSnapshot {
  summary: DiagnosticsSummary;
  entries: DiagnosticsEntry[];
}

type GoAppApi = {
  GetDiagnosticsSnapshot: () => Promise<DiagnosticsSnapshot>;
  LoadPreferences: () => Promise<Preferences>;
  SelectMediaFile: () => Promise<MediaMetadata | null>;
  UpdatePreferences: (preferences: Preferences) => Promise<Preferences>;
  ValidateMediaPath: (path: string) => Promise<MediaMetadata>;
};

declare global {
  interface Window {
    go?: {
      main?: {
        App?: Partial<GoAppApi>;
      };
    };
  }
}

export const defaultPreferences: Preferences = {
  version: 1,
  model: "Qwen3-ASR-0.6B",
  theme: "dark",
  output: {
    maxLineLength: 42,
    linesPerSubtitle: 2,
  },
  directories: {
    lastOpenDirectory: "",
    lastSaveDirectory: "",
  },
  processing: {
    alignmentChunkMinutes: 5,
  },
};

export const defaultDiagnostics: DiagnosticsSnapshot = {
  summary: {
    title: "Ready",
    message: "Logs will appear here as the app does work.",
    level: "info",
  },
  entries: [],
};

function getAppMethod<K extends keyof GoAppApi>(name: K): GoAppApi[K] {
  const method = window.go?.main?.App?.[name];
  if (typeof method !== "function") {
    return (async () => {
      throw new Error("Wails bindings are not available.");
    }) as GoAppApi[K];
  }

  return method as GoAppApi[K];
}

export function loadPreferences() {
  return getAppMethod("LoadPreferences")();
}

export function updatePreferences(preferences: Preferences) {
  return getAppMethod("UpdatePreferences")(preferences);
}

export function selectMediaFile() {
  return getAppMethod("SelectMediaFile")();
}

export function validateMediaPath(path: string) {
  return getAppMethod("ValidateMediaPath")(path);
}

export function getDiagnosticsSnapshot() {
  return getAppMethod("GetDiagnosticsSnapshot")();
}
