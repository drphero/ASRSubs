import { act, fireEvent, render, screen } from "@testing-library/react";
import { defaultModelSnapshot, defaultPreferences, defaultRuntimeReadiness } from "../../lib/backend";
import { OVERLAY_EXIT_DURATION_MS } from "../shell/useOverlayPresence";
import { SettingsDrawer } from "./SettingsDrawer";

describe("SettingsDrawer", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the drawer when open", () => {
    render(
      <SettingsDrawer
        error={null}
        modelStatuses={defaultModelSnapshot.models}
        onClose={vi.fn()}
        onDeleteModel={vi.fn()}
        onDownloadModel={vi.fn()}
        onPreferencesChange={vi.fn()}
        onPrepareRuntime={vi.fn()}
        open
        preferences={defaultPreferences}
        runtimePreparing={false}
        runtimeReadiness={defaultRuntimeReadiness}
      />,
    );

    expect(screen.getByLabelText("settings drawer")).toBeInTheDocument();
    expect(screen.getByText("Qwen3-ASR-1.7B")).toBeInTheDocument();
    expect(screen.getByLabelText("Qwen3-ASR-1.7B status")).toHaveTextContent("Not downloaded");
  });

  it("keeps the drawer mounted while closing and removes it after the exit duration", () => {
    vi.useFakeTimers();

    const { rerender } = render(
      <SettingsDrawer
        error={null}
        modelStatuses={defaultModelSnapshot.models}
        onClose={vi.fn()}
        onDeleteModel={vi.fn()}
        onDownloadModel={vi.fn()}
        onPreferencesChange={vi.fn()}
        onPrepareRuntime={vi.fn()}
        open
        preferences={defaultPreferences}
        runtimePreparing={false}
        runtimeReadiness={defaultRuntimeReadiness}
      />,
    );

    rerender(
      <SettingsDrawer
        error={null}
        modelStatuses={defaultModelSnapshot.models}
        onClose={vi.fn()}
        onDeleteModel={vi.fn()}
        onDownloadModel={vi.fn()}
        onPreferencesChange={vi.fn()}
        onPrepareRuntime={vi.fn()}
        open={false}
        preferences={defaultPreferences}
        runtimePreparing={false}
        runtimeReadiness={defaultRuntimeReadiness}
      />,
    );

    expect(screen.getByLabelText("settings drawer")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(OVERLAY_EXIT_DURATION_MS - 1);
    });
    expect(screen.getByLabelText("settings drawer")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(screen.queryByLabelText("settings drawer")).not.toBeInTheDocument();
  });

  it("applies changes immediately through the callback", () => {
    const onPreferencesChange = vi.fn();

    render(
      <SettingsDrawer
        error={null}
        modelStatuses={defaultModelSnapshot.models}
        onClose={vi.fn()}
        onDeleteModel={vi.fn()}
        onDownloadModel={vi.fn()}
        onPreferencesChange={onPreferencesChange}
        onPrepareRuntime={vi.fn()}
        open
        preferences={defaultPreferences}
        runtimePreparing={false}
        runtimeReadiness={defaultRuntimeReadiness}
      />,
    );

    fireEvent.change(screen.getByLabelText("Theme"), {
      target: { value: "light" },
    });

    expect(onPreferencesChange).toHaveBeenCalledWith({
      ...defaultPreferences,
      theme: "light",
    });
  });

  it("starts a download from the model card", () => {
    const onDownloadModel = vi.fn();

    render(
      <SettingsDrawer
        error={null}
        modelStatuses={defaultModelSnapshot.models}
        onClose={vi.fn()}
        onDeleteModel={vi.fn()}
        onDownloadModel={onDownloadModel}
        onPreferencesChange={vi.fn()}
        onPrepareRuntime={vi.fn()}
        open
        preferences={defaultPreferences}
        runtimePreparing={false}
        runtimeReadiness={defaultRuntimeReadiness}
      />,
    );

    fireEvent.click(screen.getAllByRole("button", { name: "Download model" })[0]);

    expect(onDownloadModel).toHaveBeenCalledWith("Qwen3-ASR-1.7B");
  });

  it("closes through the backdrop", () => {
    const onClose = vi.fn();

    render(
      <SettingsDrawer
        error={null}
        modelStatuses={defaultModelSnapshot.models}
        onClose={onClose}
        onDeleteModel={vi.fn()}
        onDownloadModel={vi.fn()}
        onPreferencesChange={vi.fn()}
        onPrepareRuntime={vi.fn()}
        open
        preferences={defaultPreferences}
        runtimePreparing={false}
        runtimeReadiness={defaultRuntimeReadiness}
      />,
    );

    fireEvent.click(screen.getByLabelText("Close settings"));

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("disables line controls when one-word subtitles are enabled", () => {
    render(
      <SettingsDrawer
        error={null}
        modelStatuses={defaultModelSnapshot.models}
        onClose={vi.fn()}
        onDeleteModel={vi.fn()}
        onDownloadModel={vi.fn()}
        onPreferencesChange={vi.fn()}
        onPrepareRuntime={vi.fn()}
        open
        preferences={{
          ...defaultPreferences,
          processing: {
            ...defaultPreferences.processing,
            oneWordPerSubtitle: true,
          },
        }}
        runtimePreparing={false}
        runtimeReadiness={defaultRuntimeReadiness}
      />,
    );

    expect(screen.getByLabelText("Max line length")).toBeDisabled();
    expect(screen.getByLabelText("Lines per subtitle")).toBeDisabled();
  });
});
