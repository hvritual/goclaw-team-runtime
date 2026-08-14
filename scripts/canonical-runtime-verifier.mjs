#!/usr/bin/env node
import { readFile, readdir, writeFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import net from "node:net";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import {
  buildRuntimePlan,
  validateProcessState,
  validateRecordedProcessIdentity,
} from "./runtime-selector.mjs";

const CANONICAL_CAPABILITIES = Object.freeze({
  issue_list: true,
  issue_base_detail: true,
  issue_create: true,
  issue_metadata: true,
  issue_realtime: true,
  project_resources: false,
  project_retrospectives: false,
  project_requirements: false,
  project_control: false,
  issue_detail_pull_requests: false,
  issue_timeline: false,
  issue_members: false,
  issue_reactions: false,
  issue_subscribers: false,
  issue_attachments: false,
  issue_labels: false,
  issue_properties: false,
  issue_pins: false,
  issue_children: true,
  issue_project: false,
  issue_child_progress: true,
  issue_batch: true,
  issue_acceptance: false,
});

export function expectedAcceptanceContract() {
  return {
    webURL: "http://127.0.0.1:3000",
    apiURL: "http://127.0.0.1:8000",
    grpcPort: 9000,
    legacyPort: 8080,
    workspaceSlug: "canonical-fixture",
    issueIdentifier: "CAN-1",
    email: "canonical-fixture@multica.local",
  };
}

export async function captureArtifactHashes(directory) {
  const names = (await readdir(directory))
    .filter(
      (name) =>
        name.endsWith(".db") ||
        name.endsWith(".db-wal") ||
        name.endsWith(".db-shm") ||
        name.endsWith(".db-journal") ||
        name.endsWith(".log")
    )
    .sort();
  const result = {};
  for (const name of names)
    result[name] = createHash("sha256")
      .update(await readFile(path.join(directory, name)))
      .digest("hex");
  return result;
}

export function validateProcessEvidence(state, repositoryRoot = process.cwd()) {
  if (state?.mode !== "canonical")
    throw new Error("running runtime is not canonical");
  if (
    state?.ports?.web !== 3000 ||
    state?.ports?.http !== 8000 ||
    state?.ports?.grpc !== 9000
  ) {
    throw new Error("canonical port ownership does not match 3000/8000/9000");
  }
  const expectedDatabase = path
    .resolve(repositoryRoot, "data", "multica-canonical.db")
    .toLowerCase();
  if (path.resolve(state.database ?? "").toLowerCase() !== expectedDatabase)
    throw new Error("canonical database ownership mismatch");
  const serialized = JSON.stringify(state.processes ?? []).toLowerCase();
  if (serialized.includes("legacy") || /[\\/]server[\\/]/.test(serialized)) {
    throw new Error("legacy process appears in canonical process evidence");
  }
  validateProcessState(state);
  const plan = buildRuntimePlan("canonical", repositoryRoot);
  const expected = new Map([
    [plan.backend.name, plan.backend],
    [plan.web.name, plan.web],
  ]);
  if (
    state.processes.length !== expected.size ||
    new Set(state.processes.map((item) => item.name)).size !== expected.size
  ) {
    throw new Error(
      "canonical repository process owners are incomplete or duplicated"
    );
  }
  const normalize = (value) =>
    String(value).toLowerCase().replaceAll("\\", "/");
  for (const actual of state.processes) {
    const wanted = expected.get(actual.name);
    if (
      !wanted ||
      normalize(path.resolve(actual.cwd)) !==
        normalize(path.resolve(wanted.cwd)) ||
      normalize(actual.command) !== normalize(wanted.command) ||
      JSON.stringify(actual.args.map(normalize)) !==
        JSON.stringify(wanted.args.map(normalize))
    ) {
      throw new Error(
        `${actual.name} is not bound to the current repository command/cwd/args`
      );
    }
  }
}

export function validateListenerOwnership(manifest, processes, listeners) {
  validateProcessState(manifest);
  const normalized = processes.map((item) => ({
    ...item,
    commandLine: String(item.commandLine ?? "")
      .toLowerCase()
      .replaceAll("\\", "/"),
  }));
  if (
    normalized.some(
      (item) =>
        /\/server\/(?:cmd|internal)\//.test(item.commandLine) ||
        item.commandLine.includes("cmd/sqlite-server")
    )
  ) {
    throw new Error("legacy process found during Canonical verification");
  }
  for (const spec of manifest.processes) {
    const actual = normalized.find((item) => item.pid === spec.pid);
    if (!actual)
      throw new Error(
        `owned root PID ${spec.pid} is missing from process evidence`
      );
    validateRecordedProcessIdentity(spec, actual);
  }
  const owned = new Set(manifest.pids);
  let changed = true;
  while (changed) {
    changed = false;
    for (const item of normalized) {
      if (owned.has(item.parentPid) && !owned.has(item.pid)) {
        owned.add(item.pid);
        changed = true;
      }
    }
  }
  for (const item of normalized) {
    if (
      !owned.has(item.pid) &&
      /(?:^|\s)(?:go(?:\.exe)?\s+run\s+)?(?:[^\s]*\/)?cmd\/server(?:\s|$)/.test(
        item.commandLine
      )
    ) {
      throw new Error(`non-owned cmd/server process found: PID ${item.pid}`);
    }
  }
  for (const port of [3000, 8000, 9000]) {
    const owners = listeners.filter((item) => item.port === port);
    if (owners.length !== 1 || !owned.has(owners[0].pid))
      throw new Error(`orphan or ambiguous listener ownership on port ${port}`);
  }
}

export function validateCanonicalConfig(config) {
  for (const [key, expected] of Object.entries(CANONICAL_CAPABILITIES)) {
    if (config?.feature_flags?.[key] !== expected)
      throw new Error(`Canonical capability ${key} must be ${expected}`);
  }
}

export function validateQuiescentEvidence(
  processManifest,
  listeners,
  processes = []
) {
  if (processManifest !== null)
    throw new Error(
      "rollback hash evidence requires no runtime process manifest"
    );
  const fixed = new Set([3000, 8000, 8080, 9000]);
  const active = listeners.find((item) => fixed.has(Number(item.port)));
  if (active)
    throw new Error(
      `rollback hash evidence found listener on fixed port ${active.port}`
    );
  const orphan = processes.find((item) =>
    /(?:^|\s)(?:go(?:\.exe)?\s+run\s+)?(?:[^\s]*\/)?cmd\/server(?:\s|$)/.test(
      String(item.commandLine ?? "")
        .toLowerCase()
        .replaceAll("\\", "/")
    )
  );
  if (orphan)
    throw new Error(
      `rollback hash evidence found orphan cmd/server process PID ${orphan.pid}`
    );
}

export function validateInspectionStatuses(processStatus, listenerStatus) {
  if (processStatus !== 0 || (listenerStatus !== 0 && listenerStatus !== 1)) {
    throw new Error("cannot inspect process/listener ownership");
  }
}

function systemOwnershipSnapshot() {
  if (process.platform === "win32") {
    const processResult = spawnSync(
      "powershell.exe",
      [
        "-NoProfile",
        "-Command",
        "Get-CimInstance Win32_Process | Select-Object @{n='pid';e={$_.ProcessId}},@{n='parentPid';e={$_.ParentProcessId}},@{n='commandLine';e={$_.CommandLine}},@{n='creationTime';e={$_.CreationDate.ToUniversalTime().ToString('o')}} | ConvertTo-Json -Compress",
      ],
      { encoding: "utf8", windowsHide: true }
    );
    const listenerResult = spawnSync(
      "powershell.exe",
      [
        "-NoProfile",
        "-Command",
        "Get-NetTCPConnection -State Listen | Where-Object {$_.LocalPort -in 3000,8000,8080,9000} | Select-Object @{n='port';e={$_.LocalPort}},@{n='pid';e={$_.OwningProcess}} | ConvertTo-Json -Compress",
      ],
      { encoding: "utf8", windowsHide: true }
    );
    if (processResult.status !== 0 || listenerResult.status !== 0)
      throw new Error("cannot inspect Windows process/listener ownership");
    const array = (text) => {
      const value = JSON.parse(text || "[]");
      return Array.isArray(value) ? value : [value];
    };
    return {
      processes: array(processResult.stdout),
      listeners: array(listenerResult.stdout),
    };
  }
  const processResult = spawnSync(
    "ps",
    ["-axo", "pid=,ppid=,lstart=,command="],
    { encoding: "utf8" }
  );
  const listenerResult = spawnSync(
    "lsof",
    [
      "-nP",
      "-iTCP:3000",
      "-iTCP:8000",
      "-iTCP:8080",
      "-iTCP:9000",
      "-sTCP:LISTEN",
      "-FpPn",
    ],
    { encoding: "utf8" }
  );
  validateInspectionStatuses(processResult.status, listenerResult.status);
  const processes = processResult.stdout
    .trim()
    .split(/\r?\n/)
    .filter(Boolean)
    .map((line) => {
      const match = line.trim().match(/^(\d+)\s+(\d+)\s+(.{24})\s+(.*)$/);
      return (
        match && {
          pid: Number(match[1]),
          parentPid: Number(match[2]),
          creationTime: match[3].trim(),
          commandLine: match[4],
        }
      );
    })
    .filter(Boolean);
  let pid;
  const listeners = [];
  for (const line of listenerResult.stdout.split(/\r?\n/)) {
    if (line.startsWith("p")) pid = Number(line.slice(1));
    if (line.startsWith("n")) {
      const match = line.match(/:(3000|8000|8080|9000)$/);
      if (match) listeners.push({ port: Number(match[1]), pid });
    }
  }
  return { processes, listeners };
}

async function fetchJSON(url, init) {
  const response = await fetch(url, {
    ...init,
    signal: AbortSignal.timeout(5_000),
  });
  const text = await response.text();
  let body;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = text;
  }
  if (!response.ok)
    throw new Error(
      `${init?.method ?? "GET"} ${url} failed ${response.status}: ${text}`
    );
  return body;
}

async function portOpen(port) {
  return new Promise((resolve) => {
    const socket = net.createConnection({ host: "127.0.0.1", port });
    socket.setTimeout(1_000);
    socket.once("connect", () => {
      socket.destroy();
      resolve(true);
    });
    socket.once("timeout", () => {
      socket.destroy();
      resolve(false);
    });
    socket.once("error", () => resolve(false));
  });
}

async function verifyRuntimeState(repositoryRoot) {
  const runtimeDir = path.resolve(
    process.env.MULTICA_RUNTIME_DIR ||
      path.join(repositoryRoot, ".local-runtime")
  );
  const processState = JSON.parse(
    await readFile(path.join(runtimeDir, "runtime-processes.json"), "utf8")
  );
  validateProcessEvidence(processState, repositoryRoot);
  const ownership = systemOwnershipSnapshot();
  validateListenerOwnership(
    processState,
    ownership.processes,
    ownership.listeners
  );
  for (const port of [3000, 8000, 9000]) {
    if (!(await portOpen(port)))
      throw new Error(`canonical port ${port} is not listening`);
  }
  if (await portOpen(8080))
    throw new Error("legacy backend port 8080 is still listening");
  const health = await fetchJSON("http://127.0.0.1:8000/healthz");
  const readiness = await fetchJSON("http://127.0.0.1:8000/readyz");
  if (health?.status !== "ok" || readiness?.status !== "ready")
    throw new Error("canonical health/readiness body mismatch");
  const web = await fetch("http://127.0.0.1:3000/", {
    redirect: "manual",
    signal: AbortSignal.timeout(10_000),
  });
  if (web.status < 200 || web.status >= 400)
    throw new Error(`Web root probe failed ${web.status}`);
  const proxiedConfig = await fetchJSON("http://127.0.0.1:3000/api/config");
  const directConfig = await fetchJSON("http://127.0.0.1:8000/api/config");
  if (JSON.stringify(proxiedConfig) !== JSON.stringify(directConfig)) {
    throw new Error("Web Canonical API proxy/config mismatch");
  }
  validateCanonicalConfig(proxiedConfig);
  return processState;
}

async function login() {
  const contract = expectedAcceptanceContract();
  const response = await fetchJSON(`${contract.apiURL}/auth/verify-code`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: contract.email, code: "888888" }),
  });
  if (!response?.token || response.user?.email !== contract.email)
    throw new Error("fixture login response mismatch");
  return response.token;
}

async function verifyJourney(phase) {
  const contract = expectedAcceptanceContract();
  const token = await login();
  const headers = {
    Authorization: `Bearer ${token}`,
    "X-Workspace-Slug": contract.workspaceSlug,
  };
  const issue = await fetchJSON(
    `${contract.apiURL}/api/issues/${contract.issueIdentifier}`,
    { headers }
  );
  if (issue?.identifier !== contract.issueIdentifier)
    throw new Error("fixture Issue read mismatch");
  if (phase === "mutate") {
    const value = `retained-${Date.now()}`;
    const metadata = await fetchJSON(
      `${contract.apiURL}/api/issues/${contract.issueIdentifier}/metadata/restart_readback`,
      {
        method: "PUT",
        headers: { ...headers, "Content-Type": "application/json" },
        body: JSON.stringify({ value }),
      }
    );
    if (metadata?.metadata?.restart_readback !== value)
      throw new Error("metadata mutation response mismatch");
    process.stdout.write(
      `${JSON.stringify({ phase, value, issue_id: issue.id })}\n`
    );
    return;
  }
  const expected = process.env.CANONICAL_READBACK_VALUE;
  if (!expected)
    throw new Error("CANONICAL_READBACK_VALUE is required for readback phase");
  const metadata = await fetchJSON(
    `${contract.apiURL}/api/issues/${contract.issueIdentifier}/metadata`,
    { headers }
  );
  if (metadata?.metadata?.restart_readback !== expected)
    throw new Error("retained metadata readback mismatch");
  process.stdout.write(
    `${JSON.stringify({ phase, value: expected, issue_id: issue.id })}\n`
  );
}

async function main(argv) {
  const repositoryRoot = path.resolve(
    path.dirname(fileURLToPath(import.meta.url)),
    ".."
  );
  const phase = argv[0];
  const runtimeDir = path.resolve(
    process.env.MULTICA_RUNTIME_DIR ||
      path.join(repositoryRoot, ".local-runtime")
  );
  if (phase === "snapshot" || phase === "preserved") {
    let processManifest = null;
    try {
      processManifest = JSON.parse(
        await readFile(path.join(runtimeDir, "runtime-processes.json"), "utf8")
      );
    } catch (error) {
      if (error?.code !== "ENOENT") throw error;
    }
    const ownership = systemOwnershipSnapshot();
    validateQuiescentEvidence(
      processManifest,
      ownership.listeners,
      ownership.processes
    );
    const hashes = {
      runtime: await captureArtifactHashes(runtimeDir),
      data: await captureArtifactHashes(path.join(repositoryRoot, "data")),
    };
    const filename = path.join(runtimeDir, "rollback-artifact-hashes.json");
    if (phase === "snapshot")
      await writeFile(filename, `${JSON.stringify(hashes, null, 2)}\n`);
    else if (
      JSON.stringify(hashes) !==
      JSON.stringify(JSON.parse(await readFile(filename, "utf8")))
    )
      throw new Error("rollback artifact hashes changed");
    process.stdout.write(`${JSON.stringify({ phase, hashes })}\n`);
    return;
  }
  if (phase !== "mutate" && phase !== "readback")
    throw new Error(
      "usage: canonical-runtime-verifier.mjs <mutate|readback|snapshot|preserved>"
    );
  await verifyRuntimeState(repositoryRoot);
  await verifyJourney(phase);
}

if (
  process.argv[1] &&
  path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
  main(process.argv.slice(2)).catch((error) => {
    process.stderr.write(`${error.message}\n`);
    process.exitCode = 1;
  });
}
