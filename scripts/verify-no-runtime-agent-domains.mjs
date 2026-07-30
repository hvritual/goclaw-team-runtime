#!/usr/bin/env node

import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";

const repoRoot = resolve(import.meta.dirname, "..");
const trackedFiles = execFileSync("git", ["ls-files"], {
  cwd: repoRoot,
  encoding: "utf8",
})
  .split("\n")
  .filter(Boolean);

const forbiddenPathPrefixes = [
  "packages/core/agents/",
  "packages/core/autopilots/",
  "packages/core/chat/",
  "packages/core/inbox/",
  "packages/core/runtimes/",
  "packages/core/squads/",
  "packages/views/agents/",
  "packages/views/autopilots/",
  "packages/views/chat/",
  "packages/views/inbox/",
  "packages/views/runtimes/",
  "packages/views/squads/",
  "apps/web/app/[workspaceSlug]/(dashboard)/agents/",
  "apps/web/app/[workspaceSlug]/(dashboard)/autopilots/",
  "apps/web/app/[workspaceSlug]/(dashboard)/chat/",
  "apps/web/app/[workspaceSlug]/(dashboard)/inbox/",
  "apps/web/app/[workspaceSlug]/(dashboard)/runtimes/",
  "apps/web/app/[workspaceSlug]/(dashboard)/squads/",
  "apps/docs/content/docs/agents",
  "apps/docs/content/docs/autopilots",
  "apps/docs/content/docs/chat",
  "apps/docs/content/docs/daemon-runtimes",
  "apps/docs/content/docs/inbox",
  "apps/docs/content/docs/install-agent-runtime",
  "apps/docs/content/docs/mentioning-agents",
  "apps/docs/content/docs/squads",
  "server/internal/daemon/",
  "server/internal/daemonws/",
  "server/pkg/agent/",
];

const forbiddenFiles = new Set([
  "server/pkg/db/queries/agent.sql",
  "server/pkg/db/queries/runtime.sql",
  "server/pkg/db/queries/runtime_profile.sql",
  "server/pkg/db/queries/runtime_usage.sql",
]);

const failures = [];
for (const file of trackedFiles) {
  if (!existsSync(resolve(repoRoot, file))) continue;
  if (
    forbiddenPathPrefixes.some((prefix) => file.startsWith(prefix)) ||
    forbiddenFiles.has(file)
  ) {
    failures.push(`forbidden Runtime/Agent domain file remains: ${file}`);
  }
}

const forbiddenSourceMarkers = new Map([
  [
    "server/cmd/server/router.go",
    [
      '"/api/agents"',
      '"/api/runtimes"',
      '"/api/runtime-profiles"',
      '"/api/daemon"',
      '"/ws/daemon"',
    ],
  ],
  [
    "packages/core/package.json",
    ['"./agents"', '"./runtimes"'],
  ],
  [
    "packages/views/package.json",
    ['"./agents"', '"./runtimes"'],
  ],
]);

for (const [file, markers] of forbiddenSourceMarkers) {
  const source = readFileSync(resolve(repoRoot, file), "utf8");
  for (const marker of markers) {
    if (source.includes(marker)) {
      failures.push(`${file} still contains ${marker}`);
    }
  }
}

if (failures.length > 0) {
  console.error(
    `Runtime/Agent domain removal verification failed:\n- ${failures.join("\n- ")}`,
  );
  process.exit(1);
}

console.log("PASS Runtime/Agent business-domain modules and routes are absent");
