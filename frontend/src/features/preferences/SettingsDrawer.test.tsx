import { fireEvent, render, screen } from "@testing-library/react";
import { defaultModelSnapshot, defaultPreferences } from "../../lib/backend";
import { SettingsDrawer } from "./SettingsDrawer";

describe("SettingsDrawer", () => {
  it("renders the drawer when open", () => {
    render(
      <SettingsDrawer
        error={null}
        modelStatuses={defaultModelSnapshot.models}
        onClose={vi.fn()}
        onDeleteModel={vi.fn()}
        onDownloadModel={vi.fn()}
        onPreferencesChange={vi.fn()}
        open
        preferences={defaultPreferences}
      />,
    );

    expect(screen.getByLabelText("settings drawer")).toBeInTheDocument();
    expect(screen.getByText("Qwen3-ASR-1.7B")).toBeInTheDocument();
    expect(screen.getByLabelText("Qwen3-ASR-1.7B status")).toHaveTextContent("Not downloaded");
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
        open
        preferences={defaultPreferences}
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
        open
        preferences={defaultPreferences}
      />,
    );

    fireEvent.click(screen.getAllByRole("button", { name: "Download model" })[0]);

    expect(onDownloadModel).toHaveBeenCalledWith("Qwen3-ASR-1.7B");
  });
});
