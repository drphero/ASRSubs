import { render, screen } from "@testing-library/react";
import { defaultDiagnostics } from "../../lib/backend";
import { DetailsPanel } from "./DetailsPanel";

describe("DetailsPanel", () => {
  it("shows the empty-log hint", () => {
    render(<DetailsPanel onClose={vi.fn()} open snapshot={defaultDiagnostics} />);

    expect(screen.getByText("Logs appear here during app activity.")).toBeInTheDocument();
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
});
