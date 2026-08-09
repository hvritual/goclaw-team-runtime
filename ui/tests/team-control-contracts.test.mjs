import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const root = new URL("../src/team/", import.meta.url);

async function source(name) {
  return readFile(new URL(name, root), "utf8");
}

test("three delivery lanes are exposed from the application shell", async () => {
  const [app, shell] = await Promise.all([
    readFile(new URL("../src/App.tsx", import.meta.url), "utf8"),
    source("AppShell.tsx"),
  ]);
  for (const page of ["spec", "work", "quality", "reviews", "memory"]) {
    assert.match(app, new RegExp(`["']${page}["']`));
  }
  for (const label of [
    "需求与方案",
    "任务",
    "Bug 与风险",
    "代码评审",
    "知识资产",
  ]) {
    assert.ok(shell.includes(label), `missing navigation label ${label}`);
  }
});

test("team control pages call registered project-scoped RPC methods", async () => {
  const files = await Promise.all([
    source("SpecPage.tsx"),
    source("WorkPage.tsx"),
    source("QualityPage.tsx"),
    source("ReviewPage.tsx"),
    source("TeamPage.tsx"),
  ]);
  const merged = files.join("\n");
  const required = [
    "work.items",
    "work.create",
    "work.transition",
    "issue.list",
    "assignment.list",
    "assignment.create",
    "assignment.release",
    "dev.tasks",
    "artifact.list",
    "correlation.list",
    "team.members",
    "runner.list",
    "runner.tasks",
    "repository.list",
    "repository.create",
    "document.list",
    "document.register",
    "component.list",
    "component.register",
    "policy.list",
    "policy.status",
    "policy.put",
    "delivery.projection",
    "delivery.integrity",
    "delivery.command",
  ];
  for (const method of required) {
    assert.match(merged, new RegExp(`["']${method.replace(".", "\\.")}["']`), `missing RPC ${method}`);
  }
  assert.doesNotMatch(merged, /mock|fixture|fallbackData/i);
});

test("browser credentials stay in memory and never enter web storage or URL state", async () => {
  const files = await Promise.all([
    source("client.ts"),
    source("context.tsx"),
    source("AppShell.tsx"),
  ]);
  const merged = files.join("\n");
  assert.doesNotMatch(merged, /localStorage|sessionStorage|indexedDB/i);
  assert.doesNotMatch(
    merged,
    /URLSearchParams.*token|location\.(hash|search).*token/i,
  );
  assert.match(merged, /setReviewerToken\(["']["']\)/);
  assert.match(merged, /credentials:\s*["']same-origin["']/);
});

test("quality intake preserves defect and risk as distinct event-backed aggregates", async () => {
  const quality = await source("QualityPage.tsx");
  assert.match(quality, /type Lane = ["']defect["'] \| ["']risk["']/);
  assert.match(quality, /["']defect\.create["']/);
  assert.match(quality, /["']defect\.transition["']/);
  assert.match(quality, /["']risk\.create["']/);
  assert.match(quality, /["']risk\.response\.decide["']/);
  assert.match(quality, /["']delivery\.evidence\.record["']/);
  assert.doesNotMatch(quality, /issue\.create|type:\s*issueType/);
});

test("requirement lane enforces contract, four-review, freeze, and change-intent commands", async () => {
  const spec = await source("SpecPage.tsx");
  for (const command of [
    "request.create",
    "intent.approve",
    "solution.create",
    "solution.review.decide",
    "plan.freeze",
    "change_intent.create",
  ]) {
    assert.match(spec, new RegExp(`["']${command.replaceAll(".", "\\.")}["']`));
  }
  for (const review of ["scenario", "capacity", "risk", "cost"]) {
    assert.match(spec, new RegExp(`["']${review}["']`));
  }
  assert.match(spec, /expected_revision:\s*state\.data\.projection\.revision/);
});

test("review center surfaces deterministic gates and real artifacts", async () => {
  const review = await source("ReviewPage.tsx");
  assert.match(review, /task\.last_gate/);
  assert.match(review, /artifact\.list/);
  assert.match(review, /correlation\.list/);
  assert.match(review, /DoneGate/);
});
