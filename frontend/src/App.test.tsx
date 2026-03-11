import { act, fireEvent, render, screen } from "@testing-library/react";
import App from "./App";
import * as backend from "./lib/backend";

const runtimeHandlers = new Map<string, Set<(payload: unknown) => void>>();

vi.mock("../wailsjs/runtime/runtime", () => ({
  EventsOff: vi.fn((eventName?: string) => {
    if (!eventName) {
      runtimeHandlers.clear();
      return;
    }

    runtimeHandlers.delete(eventName);
  }),
  EventsOn: vi.fn((eventName: string, handler: (payload: unknown) => void) => {
    const handlers = runtimeHandlers.get(eventName) ?? new Set<(payload: unknown) => void>();
    handlers.add(handler);
    runtimeHandlers.set(eventName, handlers);
  }),
}));

vi.mock("./lib/backend", async () => {
  const actual = await vi.importActual<typeof import("./lib/backend")>("./lib/backend");
  return {
    ...actual,
    confirmDiscardSubtitleDraft: vi.fn().mockResolvedValue(true),
    deleteModel: vi.fn(),
    getDiagnosticsSnapshot: vi.fn().mockResolvedValue(actual.defaultDiagnostics),
    getSubtitleDraft: vi.fn(),
    getTranscriptionSnapshot: vi.fn().mockResolvedValue(actual.defaultTranscriptionSnapshot),
    loadModelSnapshot: vi.fn().mockResolvedValue(actual.defaultModelSnapshot),
    loadPreferences: vi.fn().mockResolvedValue(actual.defaultPreferences),
    retryTranscription: vi.fn(),
    saveSubtitleDraft: vi.fn(),
    selectMediaFile: vi.fn(),
    startModelDownload: vi.fn(),
    startTranscription: vi.fn().mockResolvedValue({
      ...actual.defaultTranscriptionSnapshot,
      active: true,
      fileName: "clip.wav",
      filePath: "/tmp/clip.wav",
      modelID: actual.defaultPreferences.model,
      stage: "Preparing media",
    }),
    updatePreferences: vi.fn().mockResolvedValue(actual.defaultPreferences),
    validateMediaPath: vi.fn(),
  };
});

function emitRuntimeEvent(eventName: string, payload: unknown) {
  for (const handler of runtimeHandlers.get(eventName) ?? []) {
    handler(payload);
  }
}

describe("App shell", () => {
  beforeEach(() => {
    runtimeHandlers.clear();
  });

  it("defaults to the landing view with the selected model summary", async () => {
    render(<App />);

    expect(screen.getByLabelText("landing view")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Browse Media" })).toBeInTheDocument();
    expect(await screen.findByText(/Selected model: Qwen3-ASR-1.7B/i)).toBeInTheDocument();
  });

  it("opens the settings drawer", async () => {
    render(<App />);

    fireEvent.click(screen.getByRole("button", { name: "Settings" }));

    expect(await screen.findByLabelText("settings drawer")).toBeInTheDocument();
  });

  it("moves into the work view after a valid browse result", async () => {
    vi.mocked(backend.selectMediaFile).mockResolvedValueOnce({
      directory: "/tmp",
      durationLabel: "0:12",
      durationSeconds: 12,
      extension: ".wav",
      hasKnownDuration: true,
      name: "clip.wav",
      path: "/tmp/clip.wav",
      sizeBytes: 1200,
    });

    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Browse Media" }));

    expect(await screen.findByLabelText("workspace view")).toBeInTheDocument();
    expect(screen.getByText("Local transcription is set up for this file.")).toBeInTheDocument();
    expect(screen.getByText("1.2 KB")).toBeInTheDocument();
  });

  it("loads the subtitle draft once after success and preserves local edits", async () => {
    vi.mocked(backend.selectMediaFile).mockResolvedValueOnce({
      directory: "/tmp",
      durationLabel: "0:12",
      durationSeconds: 12,
      extension: ".wav",
      hasKnownDuration: true,
      name: "clip.wav",
      path: "/tmp/clip.wav",
      sizeBytes: 1200,
    });
    vi.mocked(backend.getSubtitleDraft).mockResolvedValueOnce({
      sourceFileName: "clip.wav",
      sourceFilePath: "/tmp/clip.wav",
      suggestedFilename: "clip.srt",
      text: "1\n00:00:00,000 --> 00:00:01,000\nhello world\n",
    });

    render(<App />);
    fireEvent.click(screen.getByRole("button", { name: "Browse Media" }));
    expect(await screen.findByLabelText("workspace view")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Start Transcription" }));
    expect(await screen.findByLabelText("processing view")).toBeInTheDocument();

    await act(async () => {
      emitRuntimeEvent("transcription:state", {
        active: false,
        canRetry: false,
        failedStage: "",
        failureSummary: "",
        fileName: "clip.wav",
        filePath: "/tmp/clip.wav",
        modelID: "Qwen3-ASR-1.7B",
        partCount: 0,
        partIndex: 0,
        stage: "",
      });
    });

    const textarea = await screen.findByLabelText("Editable subtitle text");
    expect(screen.getByText("Subtitle draft ready for final edits.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Run Again" })).toBeInTheDocument();
    expect(textarea).toHaveValue("1\n00:00:00,000 --> 00:00:01,000\nhello world\n");
    expect(textarea).toHaveFocus();
    expect((textarea as HTMLTextAreaElement).selectionStart).toBe(0);

    fireEvent.change(textarea, { target: { value: "edited subtitle text" } });

    await act(async () => {
      emitRuntimeEvent("transcription:state", {
        active: false,
        canRetry: false,
        failedStage: "",
        failureSummary: "",
        fileName: "clip.wav",
        filePath: "/tmp/clip.wav",
        modelID: "Qwen3-ASR-1.7B",
        partCount: 0,
        partIndex: 0,
        stage: "",
      });
    });

    expect(textarea).toHaveValue("edited subtitle text");
    expect(backend.getSubtitleDraft).toHaveBeenCalledTimes(1);
  });
});
