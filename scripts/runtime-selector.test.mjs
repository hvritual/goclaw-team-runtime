import assert from "node:assert/strict";
import { mkdtemp, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";
import { spawnSync } from "node:child_process";

import {
  applySelection,
  buildRuntimePlan,
  createLaunchFingerprint,
  launchOwnedProcesses,
  readRuntimeStatus,
  rollbackSelection,
  unlinkIfPresent,
  validateRecordedProcessIdentity,
} from "./runtime-selector.mjs";

function recordedProcess(overrides = {}) {
  const spec = {
    pid: 2147483646,
    name: "canonical-backend",
    command: "go",
    args: ["run", "./cmd/server"],
    cwd: "C:\\repo\\backend",
    creationTime: "fixture-creation-time",
    ...overrides,
  };
  return { ...spec, launchFingerprint: createLaunchFingerprint(spec) };
}

test("canonical plan freezes ports, database, and excludes legacy server commands", () => {
  const root = path.resolve("fixture-root");
  const plan = buildRuntimePlan("canonical", root);

  assert.equal(plan.web.port, 3000);
  assert.equal(plan.backend.httpPort, 8000);
  assert.equal(plan.backend.grpcPort, 9000);
  assert.equal(plan.backend.database, path.join(root, "data", "multica-canonical.db"));
  assert.equal(plan.backend.cwd, path.join(root, "backend"));
  assert.equal(plan.web.cwd, path.join(root, "apps", "web"));
  assert.equal(plan.web.command, process.execPath);
  assert.deepEqual(plan.web.args.slice(1), ["dev", "--webpack", "--hostname", "127.0.0.1", "--port", "3000"]);
  assert.equal(plan.web.args[0], path.join(root, "apps", "web", "node_modules", "next", "dist", "bin", "next"));
  assert.ok(!JSON.stringify(plan).includes(`${path.sep}server${path.sep}`));
});

test("Web launch command executes cross-platform without cmd shell wrapping", () => {
  const plan = buildRuntimePlan("canonical", process.cwd());
  const result = spawnSync(plan.web.command, [plan.web.args[0], "--help"], { encoding: "utf8", windowsHide: true });
  assert.equal(result.error, undefined);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /Next\.js|Usage/i);
});

test("owned process identity rejects PID reuse and another checkout before stop", () => {
  const spec = {
    pid: 123,
    name: "canonical-backend",
    command: "go",
    args: ["run", "./cmd/server", "-sqlite-path", "C:\\repo\\data\\multica-canonical.db"],
    cwd: "C:\\repo\\backend",
    creationTime: "20260814010203.000000+000",
  };
  spec.launchFingerprint = createLaunchFingerprint(spec);

  assert.doesNotThrow(() => validateRecordedProcessIdentity(spec, {
    creationTime: spec.creationTime,
    commandLine: '"C:\\Program Files\\Go\\bin\\go.exe" run .\\cmd\\server -sqlite-path C:\\repo\\data\\multica-canonical.db',
  }));
  assert.throws(() => validateRecordedProcessIdentity(spec, {
    creationTime: "20260814020203.000000+000",
    commandLine: 'go run .\\cmd\\server -sqlite-path C:\\repo\\data\\multica-canonical.db',
  }), /creation time/);
  assert.throws(() => validateRecordedProcessIdentity(spec, {
    creationTime: spec.creationTime,
    commandLine: 'go run .\\cmd\\server -sqlite-path D:\\other\\data\\multica-canonical.db',
  }), /command fingerprint/);
  assert.throws(() => validateRecordedProcessIdentity({ ...spec, cwd: "D:\\other\\backend" }, {
    creationTime: spec.creationTime,
    commandLine: 'go run .\\cmd\\server -sqlite-path C:\\repo\\data\\multica-canonical.db',
  }), /launch fingerprint/);
});

test("dry-run selection does not write state", async () => {
  const runtimeDir = await mkdtemp(path.join(tmpdir(), "multica-selector-"));
  const result = await applySelection("canonical", { runtimeDir, dryRun: true });

  assert.equal(result.selected, "canonical");
  assert.equal((await readRuntimeStatus(runtimeDir)).selected, null);
});

test("selection status and rollback preserve runtime artifacts", async () => {
  const runtimeDir = await mkdtemp(path.join(tmpdir(), "multica-selector-"));
  const database = path.join(runtimeDir, "multica-canonical.db");
  const log = path.join(runtimeDir, "canonical-backend.log");
  await writeFile(database, "retained-db");
  await writeFile(log, "retained-log");

  await applySelection("legacy", { runtimeDir });
  await applySelection("canonical", { runtimeDir });
  assert.deepEqual(await readRuntimeStatus(runtimeDir), {
    selected: "canonical",
    previous: "legacy",
    running: null,
  });
  await rollbackSelection({ runtimeDir });

  assert.equal((await readRuntimeStatus(runtimeDir)).selected, "legacy");
  assert.equal(await readFile(database, "utf8"), "retained-db");
  assert.equal(await readFile(log, "utf8"), "retained-log");
});

test("invalid or corrupted selector state fails closed", async () => {
  const runtimeDir = await mkdtemp(path.join(tmpdir(), "multica-selector-"));
  await assert.rejects(() => applySelection("other", { runtimeDir }), /canonical or legacy/);
  await writeFile(path.join(runtimeDir, "runtime-selector.json"), "not-json");
  await assert.rejects(() => readRuntimeStatus(runtimeDir), /invalid runtime selector state/);
});

test("status rejects malformed process manifests and does not report stale PIDs running", async () => {
  const runtimeDir = await mkdtemp(path.join(tmpdir(), "multica-selector-"));
  await applySelection("canonical", { runtimeDir });
  await writeFile(path.join(runtimeDir, "runtime-processes.json"), JSON.stringify({ mode: "other", pids: "bad" }));
  await assert.rejects(() => readRuntimeStatus(runtimeDir), /invalid runtime process state/);

  await writeFile(path.join(runtimeDir, "runtime-processes.json"), JSON.stringify({
    mode: "canonical", pids: [2147483646], processes: [recordedProcess()],
  }));
  assert.equal((await readRuntimeStatus(runtimeDir)).running, null);
});

test("status rejects process manifests whose PID ownership sets differ", async () => {
  const runtimeDir = await mkdtemp(path.join(tmpdir(), "multica-selector-"));
  await applySelection("canonical", { runtimeDir });
  await writeFile(path.join(runtimeDir, "runtime-processes.json"), JSON.stringify({
    mode: "canonical",
    pids: [process.pid, 2147483646],
    processes: [recordedProcess({ pid: process.pid, command: process.execPath })],
  }));
  await assert.rejects(() => readRuntimeStatus(runtimeDir), /PID ownership sets differ/);
});

test("selection and rollback reject a running runtime until it is explicitly stopped", async () => {
  const runtimeDir = await mkdtemp(path.join(tmpdir(), "multica-selector-"));
  await applySelection("legacy", { runtimeDir });
  await writeFile(path.join(runtimeDir, "runtime-processes.json"), JSON.stringify({
    mode: "legacy",
    pids: [process.pid],
    processes: [recordedProcess({ pid: process.pid, name: "legacy-backend", command: process.execPath, cwd: process.cwd() })],
  }));

  await assert.rejects(() => applySelection("canonical", { runtimeDir }), /stop it before selecting/);
  await assert.rejects(() => rollbackSelection({ runtimeDir }), /stop it before rollback/);
});

test("repository exposes explicit selector, lifecycle, and fixture commands", async () => {
  const makefile = await readFile("Makefile", "utf8");
  for (const target of [
    "runtime-select-canonical:",
    "runtime-select-legacy:",
    "runtime-status:",
    "runtime-start:",
    "runtime-stop:",
    "runtime-rollback:",
    "canonical-fixture:",
    "canonical-runtime-verify:",
  ]) {
    assert.ok(makefile.includes(target), `missing ${target}`);
  }
});

test("runbook freezes a reversible quiescent selector and hash workflow", async () => {
  const runbook = await readFile("backend/docs/canonical-local-runtime.md", "utf8");
  const ordered = [
    "select legacy", "select canonical", "runtime-selector.mjs stop",
    "canonical-runtime-verifier.mjs snapshot", "runtime-selector.mjs rollback",
    "canonical-runtime-verifier.mjs preserved",
  ];
  let cursor = -1;
  for (const token of ordered) {
    const next = runbook.indexOf(token, cursor + 1);
    assert.ok(next > cursor, `missing or out-of-order ${token}`);
    cursor = next;
  }
});

test("browser journey persists a sanitized HTTP and WebSocket trace artifact", async () => {
  const source = await readFile("e2e/canonical-runtime.spec.ts", "utf8");
  assert.match(source, /canonical-network-trace\.json/);
  assert.match(source, /testInfo\.attach\("canonical-network-trace"/);
  assert.match(source, /receivedEventTypes/);
  assert.doesNotMatch(source, /traceArtifact[\s\S]*authorization/i);
});

test("startup cleanup waits for every child exit before deleting ownership evidence", async () => {
  const source = await readFile("scripts/runtime-selector.mjs", "utf8");
  const wait = source.indexOf("await Promise.all(processState.processes.map((spec) => waitForExit(spec.pid)))");
  const remove = source.indexOf("await unlinkIfPresent(path.join(runtimeDir, PROCESS_FILE))", wait);
  assert.ok(wait >= 0 && remove > wait, "startup cleanup must await all exits before deleting the manifest");
});

test("concurrent stop manifest removal is idempotent", async () => {
  let removed = false;
  const racingUnlink = async () => {
    if (!removed) { removed = true; return; }
    const error = new Error("already removed");
    error.code = "ENOENT";
    throw error;
  };
  await assert.doesNotReject(() => Promise.all([
    unlinkIfPresent("runtime-processes.json", racingUnlink),
    unlinkIfPresent("runtime-processes.json", racingUnlink),
  ]));
});

test("launch cleanup stops all created children on later spawn or manifest failure", async () => {
  const plan = buildRuntimePlan("canonical", path.resolve("fixture-root"));
  const stopped = [];
  const awaited = [];
  const dependencies = {
    openLog: () => 1,
    closeLog: () => {},
    captureIdentity: async (pid) => ({ creationTime: `created-${pid}`, commandLine: "unused" }),
    stopProcess: (pid) => stopped.push(pid),
    awaitExit: async (pid) => awaited.push(pid),
  };
  let calls = 0;
  await assert.rejects(() => launchOwnedProcesses(plan, "C:\\runtime", {
    ...dependencies,
    spawnProcess: () => { calls += 1; if (calls === 2) throw new Error("second spawn failed"); return { pid: 101 }; },
    writeManifest: async () => {},
  }), /second spawn failed/);
  assert.deepEqual(stopped, [101]);
  assert.deepEqual(awaited, [101]);

  stopped.length = 0;
  awaited.length = 0;
  calls = 0;
  await assert.rejects(() => launchOwnedProcesses(plan, "C:\\runtime", {
    ...dependencies,
    spawnProcess: () => ({ pid: 201 + calls++ }),
    writeManifest: async () => { throw new Error("manifest failed"); },
  }), /manifest failed/);
  assert.deepEqual(stopped, [201, 202]);
  assert.deepEqual(awaited, [201, 202]);
});
