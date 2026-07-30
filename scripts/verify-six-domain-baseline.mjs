#!/usr/bin/env node

import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, statSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const manifestPath = resolve(repoRoot, "docs/six-domain-baseline.json");
const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
const expectedDomains = [
  "workspace",
  "member",
  "project",
  "issue",
  "task",
  "skill",
];
const expectedLayers = [
  "database",
  "backend",
  "protocol",
  "client",
  "experience",
  "cli",
  "docs",
  "tests",
];
const failures = [];

if (manifest.schema_version !== 1) {
  failures.push("manifest schema_version must be 1");
}

const domainIds = manifest.domains?.map((domain) => domain.id) ?? [];
if (JSON.stringify(domainIds) !== JSON.stringify(expectedDomains)) {
  failures.push(
    `domain order must be ${expectedDomains.join(", ")}; got ${domainIds.join(", ")}`,
  );
}

const trackedFiles = new Set(
  execFileSync("git", ["ls-files", "-z"], {
    cwd: repoRoot,
    encoding: "utf8",
  })
    .split("\0")
    .filter(Boolean),
);

for (const domain of manifest.domains ?? []) {
  for (const layer of expectedLayers) {
    const files = domain.layers?.[layer];
    if (!Array.isArray(files) || files.length === 0) {
      failures.push(`${domain.id}.${layer} must list at least one file`);
      continue;
    }

    for (const file of files) {
      if (!trackedFiles.has(file) && !existsSync(resolve(repoRoot, file))) {
        failures.push(`${domain.id}.${layer}: ${file} does not exist`);
        continue;
      }

      try {
        if (!statSync(resolve(repoRoot, file)).isFile()) {
          failures.push(`${domain.id}.${layer}: ${file} is not a regular file`);
        }
      } catch {
        failures.push(`${domain.id}.${layer}: ${file} does not exist`);
      }
    }
  }

  for (const marker of domain.markers ?? []) {
    if (!trackedFiles.has(marker.file) && !existsSync(resolve(repoRoot, marker.file))) {
      failures.push(`${domain.id}.markers: ${marker.file} does not exist`);
      continue;
    }

    const source = readFileSync(resolve(repoRoot, marker.file), "utf8");
    for (const value of marker.values ?? []) {
      if (!source.includes(value)) {
        failures.push(
          `${domain.id}.markers: ${JSON.stringify(value)} missing from ${marker.file}`,
        );
      }
    }
  }
}

if (failures.length > 0) {
  console.error(`Six-domain baseline verification failed:\n- ${failures.join("\n- ")}`);
  process.exit(1);
}

for (const domain of manifest.domains) {
  const fileCount = Object.values(domain.layers).flat().length;
  console.log(`PASS ${domain.id}: ${fileCount} tracked layer entries`);
}
console.log("PASS multica-six-domains: all six domain boundaries are present");
