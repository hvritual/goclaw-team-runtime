import { beforeEach, describe, expect, it } from "vitest";
import { useQuickCreateStore } from "./quick-create-store";

const RESET_STATE = {
  lastProjectId: null,
  keepOpen: false,
};

describe("quick create store", () => {
  beforeEach(() => {
    useQuickCreateStore.setState(RESET_STATE);
  });

  it("remembers the last project picked so frequent users skip the picker", () => {
    const { setLastProjectId } = useQuickCreateStore.getState();

    setLastProjectId("proj-1");
    expect(useQuickCreateStore.getState().lastProjectId).toBe("proj-1");

    setLastProjectId(null);
    expect(useQuickCreateStore.getState().lastProjectId).toBeNull();
  });
});
