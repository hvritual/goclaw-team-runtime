import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithI18n } from "../../test/i18n";
import { FileViewer } from "./file-viewer";

describe("FileViewer", () => {
  it("does not expose file editing when file management is unavailable", () => {
    renderWithI18n(
      <FileViewer
        path="SKILL.md"
        content=""
        editable={false}
        onChange={vi.fn()}
      />,
    );

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });
});
