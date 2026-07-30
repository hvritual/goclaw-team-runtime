// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from "vitest";
import { useIssueDraftStore } from "./draft-store";

const RESET_STATE = {
  draft: {
    shared: {
      projectId: undefined,
      priority: "none" as const,
      dueDate: null,
      attachments: [],
    },
    manual: {
      title: "",
      description: "",
      status: "todo" as const,
      startDate: null,
      assigneeType: undefined,
      assigneeId: undefined,
      labelIds: [],
      propertyValues: {},
    },
  },
  lastAssigneeType: undefined,
  lastAssigneeId: undefined,
};

describe("issue draft store", () => {
  beforeEach(() => {
    useIssueDraftStore.setState(RESET_STATE);
  });

  it("updates manual and shared fields independently", () => {
    const state = useIssueDraftStore.getState();
    state.setManual({ title: "Draft issue", assigneeType: "member" });
    state.setShared({ projectId: "project-1", priority: "high" });

    expect(useIssueDraftStore.getState().draft).toMatchObject({
      manual: { title: "Draft issue", assigneeType: "member" },
      shared: { projectId: "project-1", priority: "high" },
    });
  });

  it("prefills a new draft with the remembered member", () => {
    const state = useIssueDraftStore.getState();
    state.setLastAssignee("member", "member-1");
    state.setManual({ title: "Created issue" });
    state.clearDraft();

    expect(useIssueDraftStore.getState().draft.manual).toMatchObject({
      title: "",
      assigneeType: "member",
      assigneeId: "member-1",
    });
  });

  it("reports whether meaningful manual content exists", () => {
    expect(useIssueDraftStore.getState().hasDraft()).toBe(false);
    useIssueDraftStore.getState().setManual({ description: "Details" });
    expect(useIssueDraftStore.getState().hasDraft()).toBe(true);
  });
});
