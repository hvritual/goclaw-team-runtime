#!/usr/bin/env node
import { closeSync, existsSync, mkdirSync, openSync } from "node:fs";
import { readFile, rename, unlink, writeFile } from "node:fs/promises";
import net from "node:net";
import path from "node:path";
import { spawn, spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { fileURLToPath } from "node:url";

const STATE_FILE = "runtime-selector.json";
const PROCESS_FILE = "runtime-processes.json";
const LAST_RUN_FILE = "runtime-last-run.json";
const MODES = new Set(["canonical", "legacy"]);

function assertMode(mode) {
  if (!MODES.has(mode)) throw new Error("runtime mode must be canonical or legacy");
}

export function buildRuntimePlan(mode, repositoryRoot = process.cwd()) {
  assertMode(mode);
  const root = path.resolve(repositoryRoot);
  const webCommand = {
    name: "web",
    command: process.execPath,
    args: [path.join(root, "apps", "web", "node_modules", "next", "dist", "bin", "next"), "dev", "--webpack", "--hostname", "127.0.0.1", "--port", "3000"],
    cwd: root,
    port: 3000,
    env: {
      FRONTEND_PORT: "3000",
      REMOTE_API_URL: mode === "canonical" ? "http://127.0.0.1:8000" : "http://127.0.0.1:8080",
      NEXT_PUBLIC_API_URL: "",
      NEXT_PUBLIC_WS_URL: "",
      BACKEND_PORT: mode === "canonical" ? "8000" : "8080",
    },
  };
  if (mode === "canonical") {
    const database = path.join(root, "data", "multica-canonical.db");
    return {
      mode,
      backend: {
        name: "canonical-backend",
        command: "go",
        args: ["run", "./cmd/server", "-http-addr", "127.0.0.1:8000", "-grpc-addr", "127.0.0.1:9000", "-sqlite-path", database, "-dev-verification-code", "888888"],
        cwd: path.join(root, "backend"),
        httpPort: 8000,
        grpcPort: 9000,
        database,
        env: {},
      },
      web: webCommand,
    };
  }
  return {
    mode,
    backend: {
      name: "legacy-backend",
      command: "go",
      args: ["run", "./cmd/server"],
      cwd: path.join(root, "server"),
      httpPort: 8080,
      env: {},
    },
    web: webCommand,
  };
}

async function readJSON(filename, label, missing = null) {
  try {
    return JSON.parse(await readFile(filename, "utf8"));
  } catch (error) {
    if (error?.code === "ENOENT") return missing;
    throw new Error(`invalid ${label}: ${error.message}`);
  }
}

function validateSelectorState(value) {
  if (value == null) return null;
  if (typeof value !== "object" || !MODES.has(value.selected) || (value.previous !== null && value.previous !== undefined && !MODES.has(value.previous))) {
    throw new Error("invalid runtime selector state: unexpected shape or mode");
  }
  return value;
}

export function validateProcessState(value) {
  if (value == null) return null;
  if (typeof value !== "object" || !MODES.has(value.mode) || !Array.isArray(value.pids) || !Array.isArray(value.processes)
    || value.pids.some((pid) => !Number.isInteger(pid) || pid <= 0)
    || value.processes.some((spec) => !Number.isInteger(spec?.pid) || spec.pid <= 0 || typeof spec.name !== "string" || typeof spec.command !== "string" || !Array.isArray(spec.args) || typeof spec.cwd !== "string" || typeof spec.creationTime !== "string" || !spec.creationTime || typeof spec.launchFingerprint !== "string" || !spec.launchFingerprint)) {
    throw new Error("invalid runtime process state: unexpected shape or mode");
  }
  const manifestPIDs = [...new Set(value.pids)].sort((a, b) => a - b);
  const ownedPIDs = [...new Set(value.processes.map((spec) => spec.pid))].sort((a, b) => a - b);
  if (manifestPIDs.length !== value.pids.length || ownedPIDs.length !== value.processes.length || JSON.stringify(manifestPIDs) !== JSON.stringify(ownedPIDs)) {
    throw new Error("invalid runtime process state: PID ownership sets differ");
  }
  for (const spec of value.processes) {
    if (createLaunchFingerprint(spec) !== spec.launchFingerprint) {
      throw new Error(`invalid runtime process state: launch fingerprint mismatch for ${spec.name}`);
    }
  }
  return value;
}

async function atomicWriteJSON(filename, value) {
  mkdirSync(path.dirname(filename), { recursive: true });
  const temporary = `${filename}.${process.pid}.tmp`;
  await writeFile(temporary, `${JSON.stringify(value, null, 2)}\n`, { flag: "wx" });
  await rename(temporary, filename);
}

export async function unlinkIfPresent(filename, unlinkFile = unlink) {
  try {
    await unlinkFile(filename);
  } catch (error) {
    if (error?.code !== "ENOENT") throw error;
  }
}

export async function readRuntimeStatus(runtimeDir = path.resolve(".local-runtime")) {
  const selector = validateSelectorState(await readJSON(path.join(runtimeDir, STATE_FILE), "runtime selector state"));
  const processState = validateProcessState(await readJSON(path.join(runtimeDir, PROCESS_FILE), "runtime process state"));
  const livePids = processState?.pids.filter(pidAlive) ?? [];
  return {
    selected: selector?.selected ?? null,
    previous: selector?.previous ?? null,
    running: livePids.length > 0 ? processState.mode : null,
    ...(processState ? { stale: livePids.length === 0, pids: processState.pids, livePids } : {}),
  };
}

export async function applySelection(mode, { runtimeDir = path.resolve(".local-runtime"), dryRun = false } = {}) {
  assertMode(mode);
  const running = validateProcessState(await readJSON(path.join(runtimeDir, PROCESS_FILE), "runtime process state"));
  if (running?.pids?.some(pidAlive)) {
    throw new Error(`${running.mode} runtime is running; stop it before selecting ${mode}`);
  }
  const filename = path.join(runtimeDir, STATE_FILE);
  const current = validateSelectorState(await readJSON(filename, "runtime selector state"));
  const next = {
    selected: mode,
    previous: current?.selected && current.selected !== mode ? current.selected : current?.previous ?? null,
  };
  if (!dryRun) await atomicWriteJSON(filename, next);
  return next;
}

export async function rollbackSelection({ runtimeDir = path.resolve(".local-runtime"), dryRun = false } = {}) {
  const running = validateProcessState(await readJSON(path.join(runtimeDir, PROCESS_FILE), "runtime process state"));
  if (running?.pids?.some(pidAlive)) {
    throw new Error(`${running.mode} runtime is running; stop it before rollback`);
  }
  const filename = path.join(runtimeDir, STATE_FILE);
  const current = validateSelectorState(await readJSON(filename, "runtime selector state"));
  if (!current?.previous) throw new Error("no previous runtime selection to roll back to");
  assertMode(current.previous);
  const next = { selected: current.previous, previous: current.selected };
  if (!dryRun) await atomicWriteJSON(filename, next);
  return next;
}

function pidAlive(pid) {
  if (!Number.isInteger(pid) || pid <= 0) return false;
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

function stopPID(pid) {
  if (!pidAlive(pid)) return;
  if (process.platform === "win32") {
    spawnSync("taskkill", ["/PID", String(pid), "/T", "/F"], { stdio: "ignore" });
  } else {
    try { process.kill(-pid, "SIGTERM"); } catch { process.kill(pid, "SIGTERM"); }
  }
}

async function waitForExit(pid, timeoutMs = 5_000) {
  const deadline = Date.now() + timeoutMs;
  while (pidAlive(pid) && Date.now() < deadline) {
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  if (pidAlive(pid)) throw new Error(`owned PID ${pid} did not exit within ${timeoutMs}ms`);
}

function normalized(value) {
  return String(value ?? "").toLowerCase().replaceAll("\\", "/");
}

export function createLaunchFingerprint(spec) {
  const payload = JSON.stringify({
    command: path.resolve(spec.cwd, spec.command),
    args: spec.args,
    cwd: path.resolve(spec.cwd),
  });
  return createHash("sha256").update(normalized(payload)).digest("hex");
}

function processIdentity(pid) {
  if (process.platform === "win32") {
    const result = spawnSync("powershell.exe", [
      "-NoProfile", "-Command",
      `$p=Get-CimInstance Win32_Process -Filter \"ProcessId = ${pid}\"; if ($p) { [pscustomobject]@{commandLine=$p.CommandLine;creationTime=$p.CreationDate.ToUniversalTime().ToString('o')} | ConvertTo-Json -Compress }`,
    ], { encoding: "utf8", windowsHide: true });
    if (result.status !== 0 || !result.stdout.trim()) return null;
    try { return JSON.parse(result.stdout.trim()); } catch { return null; }
  }
  const result = spawnSync("ps", ["-p", String(pid), "-o", "lstart=", "-o", "command="], { encoding: "utf8" });
  const output = result.status === 0 ? result.stdout.trim() : "";
  if (!output) return null;
  return { creationTime: output.slice(0, 24).trim(), commandLine: output.slice(24).trim() };
}

export function validateRecordedProcessIdentity(spec, actual) {
  if (createLaunchFingerprint(spec) !== spec.launchFingerprint) {
    throw new Error(`refusing PID ${spec.pid}: recorded launch fingerprint does not match command/cwd`);
  }
  if (!actual?.creationTime || actual.creationTime !== spec.creationTime) {
    throw new Error(`refusing PID ${spec.pid}: process creation time no longer matches`);
  }
  const commandLine = normalized(actual.commandLine);
  const expectedCommand = normalized(path.basename(spec.command).replace(/\.cmd$/i, ""));
  const expectedArgs = spec.args.map(normalized);
  if (!commandLine.includes(expectedCommand) || expectedArgs.some((argument) => !commandLine.includes(argument))) {
    throw new Error(`refusing PID ${spec.pid}: command fingerprint no longer matches owned ${spec.name} process`);
  }
  return true;
}

function assertOwnedProcess(spec) {
  if (!pidAlive(spec.pid)) return false;
  return validateRecordedProcessIdentity(spec, processIdentity(spec.pid));
}

async function waitForProcessIdentity(pid, timeoutMs = 2_000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const identity = processIdentity(pid);
    if (identity?.commandLine && identity?.creationTime) return identity;
    if (!pidAlive(pid)) throw new Error(`spawned PID ${pid} exited before identity capture`);
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  throw new Error(`cannot capture identity for spawned PID ${pid}`);
}

async function portAvailable(port) {
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once("error", () => resolve(false));
    server.listen(port, "127.0.0.1", () => server.close(() => resolve(true)));
  });
}

async function waitForCanonicalReadiness(timeoutMs = 90_000) {
  const deadline = Date.now() + timeoutMs;
  let lastError = "not started";
  while (Date.now() < deadline) {
    try {
      const [healthResponse, readinessResponse] = await Promise.all([
        fetch("http://127.0.0.1:8000/healthz", { signal: AbortSignal.timeout(1_000) }),
        fetch("http://127.0.0.1:8000/readyz", { signal: AbortSignal.timeout(1_000) }),
      ]);
      const [health, readiness] = await Promise.all([healthResponse.json(), readinessResponse.json()]);
      if (healthResponse.ok && readinessResponse.ok && health.status === "ok" && readiness.status === "ready") return;
      lastError = `health=${healthResponse.status}/${health.status} readiness=${readinessResponse.status}/${readiness.status}`;
    } catch (error) {
      lastError = error.message;
    }
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error(`Canonical readiness timed out: ${lastError}`);
}

async function waitForWebReadiness(timeoutMs = 90_000) {
  const deadline = Date.now() + timeoutMs;
  let lastError = "not started";
  while (Date.now() < deadline) {
    try {
      const response = await fetch("http://127.0.0.1:3000/", { redirect: "manual", signal: AbortSignal.timeout(1_000) });
      if (response.status >= 200 && response.status < 400) return;
      lastError = `HTTP ${response.status}`;
    } catch (error) { lastError = error.message; }
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error(`Web readiness timed out: ${lastError}`);
}

export async function launchOwnedProcesses(plan, runtimeDir, dependencies = {}) {
  const spawnProcess = dependencies.spawnProcess ?? spawn;
  const captureIdentity = dependencies.captureIdentity ?? waitForProcessIdentity;
  const writeManifest = dependencies.writeManifest ?? atomicWriteJSON;
  const stopProcess = dependencies.stopProcess ?? stopPID;
  const awaitExit = dependencies.awaitExit ?? waitForExit;
  const openLog = dependencies.openLog ?? openSync;
  const closeLog = dependencies.closeLog ?? closeSync;
  const children = [];
  try {
    for (const spec of [plan.backend, plan.web]) {
      const logPath = path.join(runtimeDir, `${plan.mode}-${spec.name}.log`);
      const log = openLog(logPath, "a");
      let child;
      try {
        child = spawnProcess(spec.command, spec.args, {
          cwd: spec.cwd,
          env: { ...process.env, ...spec.env },
          detached: process.platform !== "win32",
          stdio: ["ignore", log, log],
          windowsHide: true,
        });
        if (!Number.isInteger(child?.pid) || child.pid <= 0) throw new Error(`${spec.name} did not return a valid PID`);
        children.push({ child, name: spec.name, command: spec.command, args: spec.args, cwd: spec.cwd, logPath });
      } finally {
        closeLog(log);
      }
    }
    for (const item of children) {
      const identity = await captureIdentity(item.child.pid);
      item.creationTime = identity.creationTime;
      item.launchFingerprint = createLaunchFingerprint(item);
    }
    const processState = {
      mode: plan.mode,
      supervisorPid: process.pid,
      startedAt: new Date().toISOString(),
      pids: children.map(({ child }) => child.pid),
      processes: children.map(({ child, ...item }) => ({ ...item, pid: child.pid })),
      ports: { web: plan.web.port, http: plan.backend.httpPort, grpc: plan.backend.grpcPort ?? null },
      database: plan.backend.database ?? null,
    };
    await writeManifest(path.join(runtimeDir, PROCESS_FILE), processState);
    return { children, processState };
  } catch (error) {
    for (const item of children) stopProcess(item.child.pid);
    await Promise.all(children.map((item) => awaitExit(item.child.pid)));
    throw error;
  }
}

async function startSelected({ repositoryRoot, runtimeDir, dryRun }) {
  const status = await readRuntimeStatus(runtimeDir);
  if (!status.selected) throw new Error("select canonical or legacy before start");
  const plan = buildRuntimePlan(status.selected, repositoryRoot);
  if (dryRun) return plan;
  const existing = await readJSON(path.join(runtimeDir, PROCESS_FILE), "runtime process state");
  if (existing?.pids?.some(pidAlive)) throw new Error(`${existing.mode} runtime is already running`);
  for (const port of [plan.backend.httpPort, plan.backend.grpcPort, plan.web.port].filter(Boolean)) {
    if (!(await portAvailable(port))) throw new Error(`required port ${port} is already in use`);
  }
  if (plan.mode === "canonical" && !(await portAvailable(8080))) {
    throw new Error("legacy backend port 8080 is active; stop the legacy runtime before canonical start");
  }
  mkdirSync(runtimeDir, { recursive: true });
  const { children, processState } = await launchOwnedProcesses(plan, runtimeDir);
  const startupFailure = new Promise((_, reject) => {
    for (const { child, name } of children) {
      child.once("error", (error) => reject(new Error(`${name} failed to start: ${error.message}`)));
      child.once("exit", (code, signal) => reject(new Error(`${name} exited during startup: code=${code} signal=${signal}`)));
    }
  });
  let stopping = false;
  const stop = async () => {
    if (stopping) return;
    stopping = true;
    for (const spec of processState.processes) {
      if (assertOwnedProcess(spec)) stopPID(spec.pid);
    }
    await Promise.all(processState.processes.map((spec) => waitForExit(spec.pid)));
    await atomicWriteJSON(path.join(runtimeDir, LAST_RUN_FILE), { ...processState, stoppedAt: new Date().toISOString() });
    await unlinkIfPresent(path.join(runtimeDir, PROCESS_FILE));
  };
  process.once("SIGINT", () => void stop().finally(() => process.exit(130)));
  process.once("SIGTERM", () => void stop().finally(() => process.exit(143)));
  if (plan.mode === "canonical") {
    try {
      await Promise.race([Promise.all([waitForCanonicalReadiness(), waitForWebReadiness()]), startupFailure]);
    } catch (error) {
      await stop();
      throw error;
    }
  }
  process.stdout.write(`${JSON.stringify({ started: plan.mode, ...processState }, null, 2)}\n`);
  return new Promise((resolve, reject) => {
    for (const { child, name } of children) {
      child.once("error", reject);
      child.once("exit", (code, signal) => {
        void stop().then(() => {
          if (code === 0 || signal) resolve(processState);
          else reject(new Error(`${name} exited with code ${code}`));
        }, reject);
      });
    }
  });
}

async function stopRunning(runtimeDir) {
  const filename = path.join(runtimeDir, PROCESS_FILE);
  const current = validateProcessState(await readJSON(filename, "runtime process state"));
  if (!current) return { stopped: false };
  for (const spec of current.processes ?? []) {
    if (assertOwnedProcess(spec)) {
      stopPID(spec.pid);
      await waitForExit(spec.pid);
    }
  }
  await atomicWriteJSON(path.join(runtimeDir, LAST_RUN_FILE), { ...current, stoppedAt: new Date().toISOString() });
  await unlinkIfPresent(filename);
  return { stopped: true, mode: current.mode };
}

async function main(argv) {
  const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
  const runtimeDir = path.resolve(process.env.MULTICA_RUNTIME_DIR || path.join(repositoryRoot, ".local-runtime"));
  const dryRun = argv.includes("--dry-run");
  const args = argv.filter((value) => value !== "--dry-run");
  const [command, mode] = args;
  let result;
  switch (command) {
    case "select": result = await applySelection(mode, { runtimeDir, dryRun }); break;
    case "rollback":
      if (!dryRun) await stopRunning(runtimeDir);
      result = await rollbackSelection({ runtimeDir, dryRun });
      break;
    case "status": result = await readRuntimeStatus(runtimeDir); break;
    case "plan": result = buildRuntimePlan(mode, repositoryRoot); break;
    case "start": result = await startSelected({ repositoryRoot, runtimeDir, dryRun }); break;
    case "stop": result = dryRun ? await readJSON(path.join(runtimeDir, PROCESS_FILE), "runtime process state") : await stopRunning(runtimeDir); break;
    default: throw new Error("usage: runtime-selector.mjs select <canonical|legacy> [--dry-run] | plan <mode> | start [--dry-run] | status | stop | rollback [--dry-run]");
  }
  if (result !== undefined) process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2)).catch((error) => {
    process.stderr.write(`${error.message}\n`);
    process.exitCode = 1;
  });
}
