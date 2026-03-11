import { fireEvent, render, screen } from "@testing-library/react";
import App from "./App";
import * as backend from "./lib/backend";

vi.mock("../wailsjs/runtime/runtime", () => ({
  EventsOff: vi.fn(),
  EventsOn: vi.fn(),
}));

vi.mock("./lib/backend", async () => {
  const actual = await vi.importActual<typeof import("./lib/backend")>("./lib/backend");
  return {
    ...actual,
    deleteModel: vi.fn(),
    getDiagnosticsSnapshot: vi.fn().mockResolvedValue(actual.defaultDiagnostics),
    getTranscriptionSnapshot: vi.fn().mockResolvedValue(actual.defaultTranscriptionSnapshot),
    loadModelSnapshot: vi.fn().mockResolvedValue(actual.defaultModelSnapshot),
    loadPreferences: vi.fn().mockResolvedValue(actual.defaultPreferences),
    retryTranscription: vi.fn(),
    selectMediaFile: vi.fn(),
    startModelDownload: vi.fn(),
    startTranscription: vi.fn(),
    updatePreferences: vi.fn().mockResolvedValue(actual.defaultPreferences),
    validateMediaPath: vi.fn(),
  };
});

describe("App shell", () => {
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
    expect(screen.getAllByText("0:12")).toHaveLength(2);
  });
});
