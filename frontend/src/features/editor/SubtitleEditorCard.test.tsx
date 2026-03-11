import { fireEvent, render, screen } from "@testing-library/react";
import { SubtitleEditorCard } from "./SubtitleEditorCard";
import type { SubtitleDraft } from "../../lib/backend";

const draft: SubtitleDraft = {
  text: "1\n00:00:00,000 --> 00:00:01,000\nhello world\n",
  suggestedFilename: "clip.srt",
  sourceFileName: "clip.wav",
  sourceFilePath: "/tmp/clip.wav",
};

describe("SubtitleEditorCard", () => {
  it("renders the raw subtitle text and surfaces edits", () => {
    const onChange = vi.fn();

    render(
      <SubtitleEditorCard
        draft={draft}
        dirty={false}
        focusRequestId={0}
        onChange={onChange}
        text={draft.text}
      />,
    );

    const textarea = screen.getByLabelText("Editable subtitle text");
    expect(textarea).toHaveValue(draft.text);

    fireEvent.change(textarea, { target: { value: "edited subtitle text" } });

    expect(onChange).toHaveBeenCalledWith("edited subtitle text");
  });

  it("focuses the textarea at the top when a new focus request arrives", () => {
    render(
      <SubtitleEditorCard
        draft={draft}
        dirty={false}
        focusRequestId={1}
        onChange={() => undefined}
        text={draft.text}
      />,
    );

    const textarea = screen.getByLabelText("Editable subtitle text") as HTMLTextAreaElement;
    expect(textarea).toHaveFocus();
    expect(textarea.selectionStart).toBe(0);
    expect(textarea.selectionEnd).toBe(0);
  });
});
