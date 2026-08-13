import assert from "node:assert/strict";
import test from "node:test";

import { captureArtifactHashes, expectedAcceptanceContract, validateCanonicalConfig, validateInspectionStatuses, validateListenerOwnership, validateProcessEvidence, validateQuiescentEvidence } from "./canonical-runtime-verifier.mjs";
import { createLaunchFingerprint } from "./runtime-selector.mjs";
import { mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

test("acceptance contract freezes canonical endpoints and fixture identity", () => {
  assert.deepEqual(expectedAcceptanceContract(), {
    webURL: "http://127.0.0.1:3000",
    apiURL: "http://127.0.0.1:8000",
    grpcPort: 9000,
    legacyPort: 8080,
    workspaceSlug: "canonical-fixture",
    issueIdentifier: "CAN-1",
    email: "canonical-fixture@multica.local",
  });
});

test("artifact hashes prove databases and logs are preserved", async () => {
  const root = await mkdtemp(path.join(tmpdir(), "canonical-hash-"));
  await writeFile(path.join(root, "multica-canonical.db"), "db");
  await writeFile(path.join(root, "canonical-web.log"), "log");
  await writeFile(path.join(root, "multica-canonical.db-wal"), "wal");
  await writeFile(path.join(root, "multica-canonical.db-shm"), "shm");
  await writeFile(path.join(root, "multica-canonical.db-journal"), "journal");
  const before = await captureArtifactHashes(root);
  assert.deepEqual(Object.keys(before), ["canonical-web.log", "multica-canonical.db", "multica-canonical.db-journal", "multica-canonical.db-shm", "multica-canonical.db-wal"]);
  assert.deepEqual(await captureArtifactHashes(root), before);
  await writeFile(path.join(root, "canonical-web.log"), "changed");
  assert.notDeepEqual(await captureArtifactHashes(root), before);
});

test("listener ownership accepts descendants and rejects legacy or orphan processes", () => {
  const backend = { pid: 10, name: "canonical-backend", command: "go", args: ["run", "./cmd/server", "-sqlite-path", "C:/repo/data/multica-canonical.db"], cwd: "C:/repo/backend", creationTime: "backend-start" };
  backend.launchFingerprint = createLaunchFingerprint(backend);
  const web = { pid: 20, name: "web", command: "node", args: ["C:/repo/apps/web/node_modules/next/dist/bin/next", "dev", "--port", "3000"], cwd: "C:/repo", creationTime: "web-start" };
  web.launchFingerprint = createLaunchFingerprint(web);
  const manifest = { mode: "canonical", pids: [10, 20], processes: [backend, web], ports: { web: 3000, http: 8000, grpc: 9000 } };
  const processes = [
    { pid: 10, parentPid: 1, creationTime: "backend-start", commandLine: "go run ./cmd/server -sqlite-path C:/repo/data/multica-canonical.db" },
    { pid: 11, parentPid: 10, commandLine: "canonical-server.exe" },
    { pid: 20, parentPid: 1, creationTime: "web-start", commandLine: "node C:/repo/apps/web/node_modules/next/dist/bin/next dev --port 3000" },
    { pid: 21, parentPid: 20, commandLine: "node next-server" },
  ];
  assert.doesNotThrow(() => validateListenerOwnership(manifest, processes, [{ port: 8000, pid: 11 }, { port: 9000, pid: 11 }, { port: 3000, pid: 21 }]));
  assert.throws(() => validateListenerOwnership(manifest, [...processes, { pid: 30, parentPid: 1, commandLine: "go run C:/repo/server/cmd/sqlite-server" }], []), /legacy/);
  assert.throws(() => validateListenerOwnership(manifest, processes, [{ port: 8000, pid: 99 }, { port: 9000, pid: 11 }, { port: 3000, pid: 21 }]), /orphan/);
  assert.throws(() => validateListenerOwnership(manifest, [...processes, { pid: 40, parentPid: 1, commandLine: "go run ./cmd/server -sqlite-path C:/other/data.db" }], [{ port: 8000, pid: 11 }, { port: 9000, pid: 11 }, { port: 3000, pid: 21 }]), /non-owned.*cmd\/server/);
  assert.throws(() => validateListenerOwnership(manifest, [...processes, { pid: 41, parentPid: 1, commandLine: "go run D:/other/backend/cmd/server -sqlite-path D:/other/data.db" }], [{ port: 8000, pid: 11 }, { port: 9000, pid: 11 }, { port: 3000, pid: 21 }]), /non-owned.*cmd\/server/);
  assert.throws(() => validateListenerOwnership(manifest, processes.map((item) => item.pid === 10 ? { ...item, creationTime: "reused-pid" } : item), [{ port: 8000, pid: 11 }, { port: 9000, pid: 11 }, { port: 3000, pid: 21 }]), /creation time/);
  assert.throws(() => validateListenerOwnership({ ...manifest, processes: [{ ...backend, cwd: "D:/other/backend" }, web] }, processes, [{ port: 8000, pid: 11 }, { port: 9000, pid: 11 }, { port: 3000, pid: 21 }]), /launch fingerprint/);
  assert.throws(() => validateListenerOwnership(manifest, processes.map((item) => item.pid === 10 ? { ...item, commandLine: "go run ./cmd/server -sqlite-path D:/other/data.db" } : item), [{ port: 8000, pid: 11 }, { port: 9000, pid: 11 }, { port: 3000, pid: 21 }]), /command fingerprint/);
  assert.throws(() => validateListenerOwnership({ ...manifest, pids: [10, 20, 99] }, [...processes, { pid: 99, parentPid: 1, creationTime: "forged", commandLine: "go run ./cmd/server" }], [{ port: 8000, pid: 99 }, { port: 9000, pid: 99 }, { port: 3000, pid: 21 }]), /PID ownership sets differ/);
});

test("process evidence rejects legacy or unexpected owners", () => {
  const backend = { pid: 10, name: "canonical-backend", command: "go", args: ["run", "./cmd/server", "-http-addr", "127.0.0.1:8000", "-grpc-addr", "127.0.0.1:9000", "-sqlite-path", "C:\\repo\\data\\multica-canonical.db", "-dev-verification-code", "888888"], cwd: "C:\\repo\\backend", creationTime: "backend-start" };
  backend.launchFingerprint = createLaunchFingerprint(backend);
  const web = { pid: 20, name: "web", command: process.execPath, args: ["C:\\repo\\apps\\web\\node_modules\\next\\dist\\bin\\next", "dev", "--webpack", "--hostname", "127.0.0.1", "--port", "3000"], cwd: "C:\\repo", creationTime: "web-start" };
  web.launchFingerprint = createLaunchFingerprint(web);
  const valid = {
    mode: "canonical",
    pids: [10, 20],
    ports: { web: 3000, http: 8000, grpc: 9000 },
    database: "C:\\repo\\data\\multica-canonical.db",
    processes: [backend, web],
  };
  assert.doesNotThrow(() => validateProcessEvidence(valid, "C:\\repo"));
  assert.throws(() => validateProcessEvidence({ ...valid, database: "C:\\repo\\data\\other.db" }, "C:\\repo"), /database/);
  assert.throws(() => validateProcessEvidence({ ...valid, mode: "legacy" }, "C:\\repo"), /not canonical/);
  assert.throws(() => validateProcessEvidence({ ...valid, processes: [{ name: "legacy-backend", cwd: "C:\\repo\\server", args: [] }] }, "C:\\repo"), /legacy/);
  const foreignBackend = { ...backend, args: backend.args.map((value) => value.replace("C:\\repo", "D:\\foreign")), cwd: "D:\\foreign\\backend" };
  foreignBackend.launchFingerprint = createLaunchFingerprint(foreignBackend);
  assert.throws(() => validateProcessEvidence({ ...valid, database: "D:\\foreign\\data\\multica-canonical.db", processes: [foreignBackend, web] }, "C:\\repo"), /database|repository/);
  const foreignWeb = { ...web, args: web.args.map((value) => value.replace("C:\\repo", "D:\\foreign")), cwd: "D:\\foreign" };
  foreignWeb.launchFingerprint = createLaunchFingerprint(foreignWeb);
  assert.throws(() => validateProcessEvidence({ ...valid, processes: [backend, foreignWeb] }, "C:\\repo"), /repository/);
});

test("Canonical config enforces the complete frozen capability matrix", () => {
  const feature_flags = {
    issue_list: true,
    issue_base_detail: true,
    issue_metadata: true,
    issue_realtime: true,
    issue_detail_pull_requests: false,
    issue_timeline: false,
    issue_members: false,
    issue_reactions: false,
    issue_subscribers: false,
    issue_attachments: false,
    issue_labels: false,
    issue_properties: false,
    issue_pins: false,
    issue_children: false,
    issue_project: false,
    issue_child_progress: false,
    issue_acceptance: false,
  };
  assert.doesNotThrow(() => validateCanonicalConfig({ feature_flags }));
  for (const [key, expected] of Object.entries(feature_flags)) {
    assert.throws(() => validateCanonicalConfig({ feature_flags: { ...feature_flags, [key]: !expected } }), new RegExp(key));
  }
});

test("rollback hash phases require a quiescent runtime with no fixed listeners", () => {
  assert.doesNotThrow(() => validateQuiescentEvidence(null, []));
  assert.throws(() => validateQuiescentEvidence({ mode: "canonical" }, []), /process manifest/);
  assert.throws(() => validateQuiescentEvidence(null, [{ port: 9000, pid: 44 }]), /listener.*9000/);
  assert.throws(() => validateQuiescentEvidence(null, [], [{ pid: 55, commandLine: "go run D:/other/backend/cmd/server" }]), /orphan.*cmd\/server/);
});

test("Unix ownership inspection treats lsof no-listener status as quiescent", () => {
  assert.doesNotThrow(() => validateInspectionStatuses(0, 0));
  assert.doesNotThrow(() => validateInspectionStatuses(0, 1));
  assert.throws(() => validateInspectionStatuses(1, 0), /inspect/);
  assert.throws(() => validateInspectionStatuses(0, 2), /inspect/);
});
