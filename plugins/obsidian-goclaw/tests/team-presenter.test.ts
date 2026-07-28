import {
  describe,
  expect,
  it
} from "vitest";
import { TEAM_CONTROL_RPC } from "../src/gateway-client";
import {
  issueState,
  leaseState,
  memberState,
  runnerState,
  severityState,
  teamStateClass,
  workState
} from "../src/team-presenter";

describe("team control RPC contract", () => {
  it("uses the governed team endpoint namespaces", () => {
    expect(Object.values(TEAM_CONTROL_RPC)).toEqual([
      "team.members",
      "work.items",
      "issue.list",
      "runner.list",
      "policy.status",
      "docs.summary",
      "components.summary"
    ]);
  });
});

describe("team control presentation", () => {
  it("maps operational states to concise Chinese labels and tones", () => {
    expect(memberState("active")).toEqual({ label: "在线", tone: "success" });
    expect(workState("blocked")).toEqual({ label: "受阻", tone: "danger" });
    expect(issueState("in_review")).toEqual({ label: "验证中", tone: "warning" });
    expect(runnerState("busy")).toEqual({ label: "执行中", tone: "accent" });
    expect(severityState("critical")).toEqual({ label: "致命", tone: "danger" });
    expect(teamStateClass(workState("ready"))).toBe("goclaw-team-state is-accent");
  });

  it("distinguishes healthy, expiring, expired and absent leases", () => {
    const now = Date.parse("2026-07-25T00:00:00Z");
    expect(leaseState(undefined, now)).toEqual({ label: "无租约", tone: "muted" });
    expect(leaseState("2026-07-25T00:20:00Z", now)).toEqual({
      label: "租约有效",
      tone: "success"
    });
    expect(leaseState("2026-07-25T00:04:00Z", now)).toEqual({
      label: "租约即将到期",
      tone: "warning"
    });
    expect(leaseState("2026-07-24T23:59:59Z", now)).toEqual({
      label: "租约已过期",
      tone: "danger"
    });
    expect(leaseState("not-a-date", now)).toEqual({
      label: "租约时间无效",
      tone: "danger"
    });
  });
});
