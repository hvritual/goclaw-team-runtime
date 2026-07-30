import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { openCreateIssueWithPreference } from "./create-mode-store";
import { useModalStore } from "../../modals";

describe("openCreateIssueWithPreference", () => {
  beforeEach(() => useModalStore.getState().close());
  afterEach(() => useModalStore.getState().close());

  it("opens the manual issue form", () => {
    openCreateIssueWithPreference();
    expect(useModalStore.getState().modal).toBe("create-issue");
    expect(useModalStore.getState().data).toBeNull();
  });

  it("forwards seed data", () => {
    openCreateIssueWithPreference({ project_id: "p1" });
    expect(useModalStore.getState().data).toEqual({ project_id: "p1" });
  });
});
