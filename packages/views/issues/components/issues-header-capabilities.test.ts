import { describe, expect, it } from "vitest";
import { visibleCardPropertyOptions } from "./issues-header";

describe("visibleCardPropertyOptions", () => {
  it("hides labels and sub-issue progress when their capabilities are disabled", () => {
    const keys = visibleCardPropertyOptions({
      labelsEnabled: false,
      childProgressEnabled: false,
    }).map((option) => option.key);

    expect(keys).not.toContain("labels");
    expect(keys).not.toContain("childProgress");
    expect(keys).toContain("priority");
  });
});
