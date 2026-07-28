import {
  describe,
  expect,
  it
} from "vitest";
import {
  clampPercent,
  collectionItems,
  displayInitial,
  encodeTokenProtocol,
  encodeUserTokenProtocol,
  safeExcerpt,
  shortHash
} from "../src/util";

describe("GoClaw plugin utilities", () => {
  it("encodes a UTF-8 token as a WebSocket subprotocol", () => {
    const protocol = encodeTokenProtocol("密钥-token");
    expect(protocol).toMatch(/^goclaw\.bearer\.[A-Za-z0-9_-]+$/);
    expect(protocol).not.toContain("=");
    const userProtocol = encodeUserTokenProtocol("个人-token");
    expect(userProtocol).toMatch(/^goclaw\.user\.[A-Za-z0-9_-]+$/);
    expect(userProtocol).not.toContain("=");
  });

  it("shortens hashes and user-provided excerpts", () => {
    expect(shortHash("0123456789")).toBe("01234567");
    expect(shortHash()).toBe("new file");
    expect(safeExcerpt("  a   b  ", 10)).toBe("a b");
    expect(safeExcerpt("abcdefgh", 4)).toBe("abcd…");
  });

  it("normalizes list envelopes and team display values", () => {
    expect(collectionItems([1, 2])).toEqual([1, 2]);
    expect(collectionItems({ items: ["a"] })).toEqual(["a"]);
    expect(collectionItems(undefined)).toEqual([]);
    expect(clampPercent(-20)).toBe(0);
    expect(clampPercent(61.6)).toBe(62);
    expect(clampPercent(140)).toBe(100);
    expect(clampPercent(undefined)).toBe(0);
    expect(displayInitial("  王小明")).toBe("王");
    expect(displayInitial("alice")).toBe("A");
    expect(displayInitial()).toBe("?");
  });
});
