import { fireEvent, render, screen } from "@testing-library/react";
import { MediaDropzone } from "./MediaDropzone";

describe("MediaDropzone", () => {
  it("renders browse and the selected model summary", () => {
    render(
      <MediaDropzone
        error={null}
        onBrowse={vi.fn()}
        onClearError={vi.fn()}
        selectedModel="Qwen3-ASR-0.6B"
      />,
    );

    expect(screen.getByRole("button", { name: /browse media/i })).toBeInTheDocument();
    expect(screen.getByText(/Model: Qwen3-ASR-0.6B/i)).toBeInTheDocument();
  });

  it("shows the inline error state", () => {
    render(
      <MediaDropzone
        error="This file type isn't supported."
        onBrowse={vi.fn()}
        onClearError={vi.fn()}
        selectedModel="Qwen3-ASR-1.7B"
      />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent("This file type isn't supported.");
  });

  it("highlights the dropzone during drag activity", () => {
    render(
      <MediaDropzone
        error={null}
        onBrowse={vi.fn()}
        onClearError={vi.fn()}
        selectedModel="Qwen3-ASR-0.6B"
      />,
    );

    const dropzone = screen.getByLabelText("media dropzone");
    fireEvent.dragEnter(dropzone);

    expect(dropzone.className).toContain("dropzone-active");
  });
});
