import { fireEvent, render, screen } from "@testing-library/react";
import { defaultPreferences } from "../../lib/backend";
import { SettingsDrawer } from "./SettingsDrawer";

describe("SettingsDrawer", () => {
  it("renders the drawer when open", () => {
    render(
      <SettingsDrawer
        error={null}
        onClose={vi.fn()}
        onPreferencesChange={vi.fn()}
        open
        preferences={defaultPreferences}
      />,
    );

    expect(screen.getByLabelText("settings drawer")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Qwen3-ASR-0.6B")).toBeInTheDocument();
  });

  it("applies changes immediately through the callback", () => {
    const onPreferencesChange = vi.fn();

    render(
      <SettingsDrawer
        error={null}
        onClose={vi.fn()}
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
});
