import { describe, expect, it } from "vitest";
import { availableCreateMethods } from "./create-skill-dialog";

describe("availableCreateMethods", () => {
  it("hides import until the skill_import capability is installed", () => {
    expect(availableCreateMethods(false)).toEqual(["manual"]);
    expect(availableCreateMethods(true)).toEqual(["manual", "archive", "url"]);
  });
});
