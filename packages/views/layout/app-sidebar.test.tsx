import { describe, expect, it } from "vitest";
import { isNavActive, isPinReorderEnabled } from "./app-sidebar";

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

describe("isPinReorderEnabled", () => {
  it("requires loaded explicit installation evidence", () => {
    expect(isPinReorderEnabled(false, { pin_reorder: true })).toBe(false);
    expect(isPinReorderEnabled(true, {})).toBe(false);
    expect(isPinReorderEnabled(true, { pin_reorder: false })).toBe(false);
    expect(isPinReorderEnabled(true, { pin_reorder: true })).toBe(true);
  });
});
