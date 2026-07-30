import { describe, expect, it } from "vitest";
import { isNavActive } from "./app-sidebar";

describe("isNavActive", () => {
  it("matches a workspace section and its detail pages", () => {
    expect(isNavActive("/acme/issues", "/acme/issues")).toBe(true);
    expect(isNavActive("/acme/issues/MUL-12", "/acme/issues")).toBe(true);
    expect(isNavActive("/acme/tasks/task-1", "/acme/tasks")).toBe(true);
  });

  it("does not match sibling sections or lookalike prefixes", () => {
    expect(isNavActive("/acme/projects", "/acme/issues")).toBe(false);
    expect(isNavActive("/acme/issues-archive", "/acme/issues")).toBe(false);
  });
});
