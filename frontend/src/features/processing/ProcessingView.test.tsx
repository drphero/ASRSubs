import { render, screen } from "@testing-library/react";
import { defaultModelSnapshot, defaultTranscriptionSnapshot } from "../../lib/backend";
import { ProcessingView } from "./ProcessingView";

describe("ProcessingView", () => {
  it("shows the active stage with file and model context", () => {
    render(
      <ProcessingView
        selectedModelStatus={defaultModelSnapshot.models[0]}
        snapshot={{
          ...defaultTranscriptionSnapshot,
          active: true,
          fileName: "clip.wav",
          filePath: "/tmp/clip.wav",
          modelID: "Qwen3-ASR-1.7B",
          stage: "Transcribing",
        }}
      />,
    );

    expect(screen.getByLabelText("processing view")).toBeInTheDocument();
    expect(screen.getByText("Transcribing")).toBeInTheDocument();
    expect(screen.getByText("clip.wav")).toBeInTheDocument();
    expect(screen.getByText("Qwen3-ASR-1.7B")).toBeInTheDocument();
  });

  it("shows chunk progress when part counters are available", () => {
    render(
      <ProcessingView
        selectedModelStatus={defaultModelSnapshot.models[0]}
        snapshot={{
          ...defaultTranscriptionSnapshot,
          active: true,
          fileName: "feature.wav",
          filePath: "/tmp/feature.wav",
          modelID: "Qwen3-ASR-1.7B",
          stage: "Aligning",
          partIndex: 2,
          partCount: 4,
        }}
      />,
    );

    expect(screen.getByText("Part 2 of 4")).toBeInTheDocument();
  });

  it("shows the active download target while downloading a model", () => {
    render(
      <ProcessingView
        selectedModelStatus={defaultModelSnapshot.models[0]}
        snapshot={{
          ...defaultTranscriptionSnapshot,
          active: true,
          downloadTargetName: "Qwen3-ForcedAligner-0.6B",
          fileName: "feature.wav",
          filePath: "/tmp/feature.wav",
          modelID: "Qwen3-ASR-1.7B",
          stage: "Downloading model",
        }}
      />,
    );

    expect(screen.getByText("Qwen3-ForcedAligner-0.6B")).toBeInTheDocument();
  });
});
