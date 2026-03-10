import { render, screen } from "@testing-library/react";
import App from "./App";

describe("App shell", () => {
  it("defaults to the landing view with the selected model summary", () => {
    render(<App />);

    expect(screen.getByLabelText("landing view")).toBeInTheDocument();
    expect(screen.getByLabelText("selected model")).toHaveTextContent("Qwen3-ASR-0.6B");
    expect(screen.getByRole("button", { name: "Browse Media" })).toBeInTheDocument();
  });
});
