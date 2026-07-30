import { describe, expect, it } from "vitest";
import { taskDetailOptions, taskKeys, taskListOptions } from "./queries";

describe("task queries", () => {
  it("scopes list and detail keys to the workspace", () => {
    expect(taskKeys.list("workspace-a")).toEqual([
      "tasks",
      "workspace-a",
      "list",
    ]);
    expect(taskKeys.detail("workspace-a", "task-1")).toEqual([
      "tasks",
      "workspace-a",
      "detail",
      "task-1",
    ]);
  });

  it("disables an empty task detail request", () => {
    expect(taskDetailOptions("workspace-a", "").enabled).toBe(false);
    expect(taskListOptions("workspace-a").queryKey).toEqual(
      taskKeys.list("workspace-a"),
    );
  });
});
