export type ThemeMode = "dark" | "light";
export type ModelID = "Qwen3-ASR-0.6B" | "Qwen3-ASR-1.7B";
export type ModelDownloadState = "not_downloaded" | "downloading" | "ready" | "failed";

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
  oneWordPerSubtitle: boolean;
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

export interface RuntimeReadiness {
  state: "ready" | "missing" | "failed" | string;
  rootDir: string;
  pythonPath: string;
  workerPath: string;
  detail: string;
}

export interface ModelStatus {
  id: ModelID;
  name: string;
  repoId: string;
  description: string;
  speedDescription: string;
  qualityDescription: string;
  systemRequirement: string;
  default: boolean;
  state: ModelDownloadState;
  stateLabel: "Not downloaded" | "Downloading" | "Ready" | "Failed";
  path: string;
  error?: string;
}

export interface ModelSnapshot {
  version: number;
  models: ModelStatus[];
}

export interface StartTranscriptionRequest {
  mediaPath: string;
  modelID: ModelID;
}

export interface SubtitleDraft {
  text: string;
  suggestedFilename: string;
  sourceFilePath: string;
  sourceFileName: string;
}

export interface SubtitleValidationIssue {
  line: number;
  message: string;
}

export interface SaveSubtitleDraftRequest {
  text: string;
  suggestedFilename: string;
}

export interface SaveSubtitleDraftResult {
  status: "saved" | "canceled" | "invalid";
  path?: string;
  fileName?: string;
  validationIssue?: SubtitleValidationIssue;
}

export type TranscriptionStage =
  | ""
  | "Preparing media"
  | "Downloading model"
  | "Transcribing"
  | "Aligning"
  | "Building subtitles";

export interface TranscriptionSnapshot {
  active: boolean;
  canRetry: boolean;
  stage: TranscriptionStage;
  downloadTargetName?: string;
  failedStage: TranscriptionStage;
  partIndex: number;
  partCount: number;
  filePath: string;
  fileName: string;
  modelID: ModelID | "";
  failureSummary: string;
}

type GoAppApi = {
  DeleteModel: (modelID: ModelID) => Promise<ModelStatus>;
  ConfirmDiscardSubtitleDraft: () => Promise<boolean>;
  EnsureRuntimeReady: () => Promise<RuntimeReadiness>;
  GetModelState: (modelID: ModelID) => Promise<ModelStatus>;
  GetDiagnosticsSnapshot: () => Promise<DiagnosticsSnapshot>;
  GetRuntimeReadiness: () => Promise<RuntimeReadiness>;
  GetSubtitleDraft: () => Promise<SubtitleDraft>;
  GetTranscriptionSnapshot: () => Promise<TranscriptionSnapshot>;
  LoadPreferences: () => Promise<Preferences>;
  LoadModelSnapshot: () => Promise<ModelSnapshot>;
  RetryTranscription: () => Promise<TranscriptionSnapshot>;
  SaveSubtitleDraft: (request: SaveSubtitleDraftRequest) => Promise<SaveSubtitleDraftResult>;
  SelectMediaFile: () => Promise<MediaMetadata | null>;
  StartModelDownload: (modelID: ModelID) => Promise<ModelStatus>;
  StartTranscription: (request: StartTranscriptionRequest) => Promise<TranscriptionSnapshot>;
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
  model: "Qwen3-ASR-1.7B",
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
    oneWordPerSubtitle: false,
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

export const defaultRuntimeReadiness: RuntimeReadiness = {
  state: "missing",
  rootDir: "",
  pythonPath: "",
  workerPath: "",
  detail: "Managed runtime has not been prepared yet.",
};

const defaultModelStatuses: ModelStatus[] = [
  {
    id: "Qwen3-ASR-1.7B",
    name: "Qwen3-ASR-1.7B",
    repoId: "Qwen/Qwen3-ASR-1.7B",
    description: "Highest accuracy for mixed speech at the cost of a heavier local runtime footprint.",
    speedDescription: "Slower, best quality",
    qualityDescription: "Best accuracy on harder audio",
    systemRequirement: "More RAM and longer first download",
    default: true,
    state: "not_downloaded",
    stateLabel: "Not downloaded",
    path: "",
  },
  {
    id: "Qwen3-ASR-0.6B",
    name: "Qwen3-ASR-0.6B",
    repoId: "Qwen/Qwen3-ASR-0.6B",
    description: "Faster startup with a lighter model footprint for smaller machines and quick checks.",
    speedDescription: "Faster, lighter runtime",
    qualityDescription: "Good quality for quicker runs",
    systemRequirement: "Lower memory use",
    default: false,
    state: "not_downloaded",
    stateLabel: "Not downloaded",
    path: "",
  },
];

export const defaultModelSnapshot: ModelSnapshot = {
  version: 1,
  models: defaultModelStatuses,
};

export const defaultTranscriptionSnapshot: TranscriptionSnapshot = {
  active: false,
  canRetry: false,
  stage: "",
  downloadTargetName: "",
  failedStage: "",
  partIndex: 0,
  partCount: 0,
  filePath: "",
  fileName: "",
  modelID: "",
  failureSummary: "",
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

export function getRuntimeReadiness() {
  return getAppMethod("GetRuntimeReadiness")();
}

export function ensureRuntimeReady() {
  return getAppMethod("EnsureRuntimeReady")();
}

export function loadModelSnapshot() {
  return getAppMethod("LoadModelSnapshot")();
}

export function getModelState(modelID: ModelID) {
  return getAppMethod("GetModelState")(modelID);
}

export function startModelDownload(modelID: ModelID) {
  return getAppMethod("StartModelDownload")(modelID);
}

export function deleteModel(modelID: ModelID) {
  return getAppMethod("DeleteModel")(modelID);
}

export function getTranscriptionSnapshot() {
  return getAppMethod("GetTranscriptionSnapshot")();
}

export function getSubtitleDraft() {
  return getAppMethod("GetSubtitleDraft")();
}

export function startTranscription(request: StartTranscriptionRequest) {
  return getAppMethod("StartTranscription")(request);
}

export function retryTranscription() {
  return getAppMethod("RetryTranscription")();
}

export function saveSubtitleDraft(request: SaveSubtitleDraftRequest) {
  return getAppMethod("SaveSubtitleDraft")(request);
}

export function confirmDiscardSubtitleDraft() {
  return getAppMethod("ConfirmDiscardSubtitleDraft")();
}
