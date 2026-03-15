import { act, fireEvent, render, screen } from "@testing-library/react";
import { defaultDiagnostics } from "../../lib/backend";
import { OVERLAY_EXIT_DURATION_MS } from "../shell/useOverlayPresence";
import { DetailsPanel } from "./DetailsPanel";

describe("DetailsPanel", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows the empty-log hint", () => {
    render(<DetailsPanel onClose={vi.fn()} open snapshot={defaultDiagnostics} />);

    expect(screen.getByText("Logs appear here during app activity.")).toBeInTheDocument();
  });

  it("stays mounted during the close animation before unmounting", () => {
    vi.useFakeTimers();

    const { rerender } = render(<DetailsPanel onClose={vi.fn()} open snapshot={defaultDiagnostics} />);

    rerender(<DetailsPanel onClose={vi.fn()} open={false} snapshot={defaultDiagnostics} />);

    expect(screen.getByLabelText("details panel")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(OVERLAY_EXIT_DURATION_MS - 1);
    });
    expect(screen.getByLabelText("details panel")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(screen.queryByLabelText("details panel")).not.toBeInTheDocument();
  });

  it("renders entries below the status summary", () => {
    render(
      <DetailsPanel
        onClose={vi.fn()}
        open
        snapshot={{
          summary: {
            level: "warning",
            message: "This file type isn't supported.",
            title: "Check this state",
          },
          entries: [
            {
              id: "1",
              level: "warning",
              message: "This file type isn't supported.",
              source: "intake",
              timestamp: "2026-03-11T00:00:00Z",
            },
          ],
        }}
      />,
    );

    expect(screen.getByText("Check this state")).toBeInTheDocument();
    expect(screen.getByText("intake")).toBeInTheDocument();
  });

  it("closes through the backdrop", () => {
    const onClose = vi.fn();

    render(<DetailsPanel onClose={onClose} open snapshot={defaultDiagnostics} />);

    fireEvent.click(screen.getByLabelText("Close details"));

    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
