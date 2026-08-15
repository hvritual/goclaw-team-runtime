import { expect, test, type APIResponse, type Page } from "@playwright/test";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const WEB = "http://127.0.0.1:3000";
const EMAIL = "canonical-fixture@multica.local";
const SLUG = "canonical-fixture";
const ISSUE = "CAN-1";
const ISSUE_ID = "01990000-0000-7000-8000-000000000004";
const C8_PHASE = process.env.C8_PHASE;
const C8_EVIDENCE_DIR = process.env.C8_EVIDENCE_DIR;
const C8_CANDIDATE = process.env.C8_CANDIDATE;
const C9_PHASE = process.env.C9_PHASE;
const C9_EVIDENCE_DIR = process.env.C9_EVIDENCE_DIR;
const C9_CANDIDATE = process.env.C9_CANDIDATE;

interface AttachmentArtifact {
  id: string;
  filename: string;
  content: string;
  sizeBytes: number;
}

interface AttachmentEvidence extends AttachmentArtifact {
  afterRestart?: AttachmentArtifact;
  cleanupIds: string[];
}

function attachmentEvidenceDir() {
  if (!C8_EVIDENCE_DIR) throw new Error("C8_EVIDENCE_DIR is required");
  return C8_EVIDENCE_DIR;
}

function attachmentCandidate() {
  if (!C8_CANDIDATE) throw new Error("C8_CANDIDATE is required");
  return C8_CANDIDATE;
}

function c9EvidenceDir() {
  if (!C9_EVIDENCE_DIR) throw new Error("C9_EVIDENCE_DIR is required");
  return C9_EVIDENCE_DIR;
}

function c9Candidate() {
  if (!C9_CANDIDATE) throw new Error("C9_CANDIDATE is required");
  return C9_CANDIDATE;
}

async function browserIdentity(page: Page) {
  const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;
  if (!executablePath) {
    throw new Error(
      "PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH is required for C8 evidence"
    );
  }
  return {
    executablePath: path.resolve(executablePath),
    userAgent: await page.evaluate(() => navigator.userAgent),
  };
}

async function loginFixture(page: Page) {
  await page.goto(`${WEB}/login`, { waitUntil: "domcontentloaded" });
  await page.waitForLoadState("networkidle");
  await page.locator("#login-email").fill(EMAIL);
  await page.getByRole("button", { name: "Continue" }).click();
  await page.locator('input[autocomplete="one-time-code"]').fill("888888");
  await page.waitForURL(new RegExp(`/${SLUG}/issues`));
}

async function openFixtureIssue(page: Page) {
  await page.goto(`${WEB}/${SLUG}/issues/${ISSUE_ID}`, {
    waitUntil: "domcontentloaded",
  });
  await expect(
    page.getByText("Canonical runtime acceptance", { exact: true })
  ).toBeVisible({ timeout: 30_000 });
}

async function uploadFromDescriptionControl(page: Page, filename: string) {
  const chooserPromise = page.waitForEvent("filechooser");
  await page.getByRole("button", { name: "Attach file" }).first().click();
  const chooser = await chooserPromise;
  await chooser.setFiles(filename);
}

async function awaitCookieRealtimeSubscription(page: Page, wsFrames: string[]) {
  const csrfCookie = (await page.context().cookies(WEB)).find(
    (cookie) => cookie.name === "multica_csrf"
  );
  expect(csrfCookie).toBeDefined();
  const headers = {
    "X-Workspace-Slug": SLUG,
    "X-CSRF-Token": csrfCookie!.value,
  };
  const key = "c8_attachment_ws_ready";
  let attempts = 0;
  let observed = false;

  try {
    // Cookie-mode sockets intentionally receive no auth_ack. A bounded,
    // self-cleaning metadata event therefore proves that the server has added
    // this browser to the Workspace subscription before the measured upload.
    for (let attempt = 1; attempt <= 10 && !observed; attempt += 1) {
      attempts = attempt;
      const marker = `ready-${Date.now()}-${attempt}`;
      const response = await page.request.put(
        `${WEB}/api/issues/${ISSUE}/metadata/${key}`,
        { headers, data: { value: marker } }
      );
      expect(response.status()).toBe(200);
      for (let tick = 0; tick < 10 && !observed; tick += 1) {
        observed = wsFrames.some(
          (frame) =>
            frame.includes("issue_metadata:changed") && frame.includes(marker)
        );
        if (!observed) await page.waitForTimeout(25);
      }
    }
  } finally {
    const cleanup = await page.request.delete(
      `${WEB}/api/issues/${ISSUE}/metadata/${key}`,
      { headers }
    );
    expect(cleanup.status()).toBe(200);
  }

  expect(observed).toBe(true);
  return attempts;
}

function sanitizeAttachmentTrace(
  values: Array<{ method: string; url: string; status: number }>
) {
  return values.map(({ method, url, status }) => {
    const parsed = new URL(url);
    return { method, origin: parsed.origin, path: parsed.pathname, status };
  });
}

test("Canonical new user creates the first Workspace and completes onboarding", async ({
  page,
}, testInfo) => {
  const runId = Date.now();
  const email = `onboarding-${runId}@example.com`;
  const workspaceName = `Onboarding ${runId}`;
  const slug = `onboarding-${runId}`;
  const responses: Array<{ method: string; path: string; status: number }> = [];
  page.on("response", (response) => {
    const request = response.request();
    const url = new URL(response.url());
    responses.push({
      method: request.method(),
      path: url.pathname,
      status: response.status(),
    });
  });

  await page.goto(`${WEB}/login`, { waitUntil: "domcontentloaded" });
  await page.waitForLoadState("networkidle");
  await page.locator("#login-email").fill(email);
  await page.getByRole("button", { name: "Continue" }).click();
  await page.locator('input[autocomplete="one-time-code"]').fill("888888");
  await page.waitForURL(`${WEB}/onboarding`);

  await page.locator("#ws-name").fill(workspaceName);
  await expect(page.locator("#ws-slug")).toHaveValue(slug);
  const createButton = page.getByRole("button", {
    name: `Create ${workspaceName}`,
  });
  await expect(createButton).toBeEnabled();
  await createButton.click();
  await page.waitForURL(new RegExp(`/${slug}/issues`));
  await page.reload({ waitUntil: "domcontentloaded" });
  await expect(page).toHaveURL(new RegExp(`/${slug}/issues`));

  expect(responses).toContainEqual({
    method: "POST",
    path: "/api/workspaces",
    status: 201,
  });
  expect(responses).toContainEqual({
    method: "POST",
    path: "/api/me/onboarding/complete",
    status: 200,
  });
  const tracePath = testInfo.outputPath("canonical-onboarding-trace.json");
  await writeFile(
    tracePath,
    `${JSON.stringify({ email, slug, responses }, null, 2)}\n`
  );
  await testInfo.attach("canonical-onboarding-trace", {
    path: tracePath,
    contentType: "application/json",
  });
});

test("Canonical UI login, Workspace, Issue and metadata journey has no legacy traffic", async ({
  page,
}, testInfo) => {
  const httpTrace: Array<{ method: string; url: string }> = [];
  const wsTrace: string[] = [];
  const wsFrames: string[] = [];
  page.on("request", (request) =>
    httpTrace.push({ method: request.method(), url: request.url() })
  );
  page.on("websocket", (socket) => {
    wsTrace.push(socket.url());
    socket.on("framereceived", (event) => wsFrames.push(String(event.payload)));
  });

  await page.goto(`${WEB}/login`, { waitUntil: "domcontentloaded" });
  await page.waitForLoadState("networkidle");
  const emailInput = page.locator("#login-email");
  await emailInput.fill(EMAIL);
  await expect(emailInput).toHaveValue(EMAIL);
  const continueButton = page.getByRole("button", { name: "Continue" });
  await expect(continueButton).toBeEnabled();
  await continueButton.click();
  await page.locator('input[autocomplete="one-time-code"]').fill("888888");
  await page.waitForURL(new RegExp(`/${SLUG}/issues`));

  await expect
    .poll(() => httpTrace.some(({ url }) => url.includes("/api/workspaces")))
    .toBe(true);
  await page.goto(`${WEB}/${SLUG}/issues`, { waitUntil: "domcontentloaded" });
  await page.getByText("Canonical runtime acceptance", { exact: true }).click();
  await page.waitForURL(new RegExp(`/${SLUG}/issues/(?:${ISSUE}|0199)`));
  await expect(
    page.getByRole("button", {
      name: "Canonical runtime acceptance",
      exact: true,
    }),
  ).toBeVisible();

  const value = `browser-${Date.now()}`;
  const metadataTrace = await page.evaluate(
    async ({ issue, value }) => {
      const csrf = document.cookie
        .split("; ")
        .find((item) => item.startsWith("multica_csrf="))
        ?.split("=")[1];
      const headers = {
        "Content-Type": "application/json",
        "X-Workspace-Slug": "canonical-fixture",
        ...(csrf ? { "X-CSRF-Token": decodeURIComponent(csrf) } : {}),
      };
      const urls: string[] = [];
      const call = async (method: string, suffix = "", body?: string) => {
        const url = `/api/issues/${issue}/metadata${suffix}`;
        urls.push(`${method} ${url}`);
        const response = await fetch(url, { method, headers, body });
        if (!response.ok)
          throw new Error(
            `${method} ${url}: ${response.status} ${await response.text()}`
          );
        return response.json();
      };
      await call("GET");
      const put = await call(
        "PUT",
        "/browser_readback",
        JSON.stringify({ value })
      );
      if (put.metadata.browser_readback !== value)
        throw new Error("PUT readback mismatch");
      const get = await call("GET");
      if (get.metadata.browser_readback !== value)
        throw new Error("GET readback mismatch");
      return urls;
    },
    { issue: ISSUE, value }
  );
  expect(metadataTrace).toEqual([
    `GET /api/issues/${ISSUE}/metadata`,
    `PUT /api/issues/${ISSUE}/metadata/browser_readback`,
    `GET /api/issues/${ISSUE}/metadata`,
  ]);
  await expect
    .poll(() =>
      wsFrames.some(
        (frame) =>
          frame.includes("issue_metadata:changed") && frame.includes(value)
      )
    )
    .toBe(true);
  await expect(page.getByTestId("issue-base-detail")).toBeVisible();
  await expect(page.getByText("Canonical runtime acceptance")).toBeVisible();
  const removed = await page.evaluate(async (issue) => {
    const csrf = document.cookie
      .split("; ")
      .find((item) => item.startsWith("multica_csrf="))
      ?.split("=")[1];
    const headers = {
      "X-Workspace-Slug": "canonical-fixture",
      ...(csrf ? { "X-CSRF-Token": decodeURIComponent(csrf) } : {}),
    };
    const deleted = await fetch(
      `/api/issues/${issue}/metadata/browser_readback`,
      { method: "DELETE", headers }
    );
    if (!deleted.ok) throw new Error(`DELETE failed ${deleted.status}`);
    const response = await fetch(`/api/issues/${issue}/metadata`, { headers });
    return response.json();
  }, ISSUE);
  expect(Object.hasOwn(removed.metadata, "browser_readback")).toBe(false);

  expect(httpTrace.some(({ url }) => url.includes("/auth/send-code"))).toBe(
    true
  );
  expect(httpTrace.some(({ url }) => url.includes("/auth/verify-code"))).toBe(
    true
  );
  expect(
    httpTrace.some(({ url }) => url.includes(`/api/issues/${ISSUE}`))
  ).toBe(true);
  expect(
    [...httpTrace.map(({ url }) => url), ...wsTrace].every(
      (url) => new URL(url).port !== "8080"
    )
  ).toBe(true);
  expect(
    wsTrace.some(
      (url) => new URL(url).port === "3000" || new URL(url).port === "8000"
    )
  ).toBe(true);
  await expect
    .poll(
      () =>
        wsFrames.filter((frame) => frame.includes("issue_metadata:changed"))
          .length
    )
    .toBeGreaterThanOrEqual(2);

  const receivedEventTypes = wsFrames
    .map((frame) => {
      try {
        return JSON.parse(frame)?.type;
      } catch {
        return "unparseable";
      }
    })
    .filter((type): type is string => typeof type === "string");
  const traceArtifact = {
    http: httpTrace.map(({ method, url }) => {
      const parsed = new URL(url);
      return { method, origin: parsed.origin, path: parsed.pathname };
    }),
    websockets: wsTrace.map((url) => {
      const parsed = new URL(url);
      return { origin: parsed.origin, path: parsed.pathname };
    }),
    metadataOperations: metadataTrace,
    receivedEventTypes,
  };
  const tracePath = testInfo.outputPath("canonical-network-trace.json");
  await writeFile(tracePath, `${JSON.stringify(traceArtifact, null, 2)}\n`);
  await testInfo.attach("canonical-network-trace", {
    path: tracePath,
    contentType: "application/json",
  });
});

test("Canonical Projects page loads and its visible actions do not request missing routes", async ({
  page,
}, testInfo) => {
  const responses: Array<{ method: string; path: string; status: number }> = [];
  const wsFrames: string[] = [];
  page.on("response", (response) => {
    const request = response.request();
    const url = new URL(response.url());
    if (url.origin === WEB)
      responses.push({
        method: request.method(),
        path: url.pathname,
        status: response.status(),
      });
  });
  page.on("websocket", (socket) => {
    socket.on("framereceived", (event) => wsFrames.push(String(event.payload)));
  });

  const login = await page.request.post(`${WEB}/auth/verify-code`, {
    data: { email: EMAIL, code: "888888" },
  });
  expect(login.status()).toBe(200);
  const csrfCookie = (await page.context().cookies(WEB)).find(
    (cookie) => cookie.name === "multica_csrf"
  );
  const apiHeaders = {
    "X-Workspace-Slug": SLUG,
    ...(csrfCookie ? { "X-CSRF-Token": csrfCookie.value } : {}),
  };
  const previousProjects = await page.request.get(`${WEB}/api/projects`, {
    headers: apiHeaders,
  });
  if (previousProjects.ok()) {
    const body = (await previousProjects.json()) as {
      projects?: Array<{ id: string; title: string }>;
    };
    for (const project of body.projects ?? []) {
      if (project.title.startsWith("Browser Project ")) {
        await page.request.delete(`${WEB}/api/projects/${project.id}`, {
          headers: apiHeaders,
        });
      }
    }
  }

  await page.goto(`${WEB}/${SLUG}/projects`, { waitUntil: "domcontentloaded" });
  await expect(
    page.getByRole("heading", { name: "Projects", exact: true })
  ).toBeVisible({ timeout: 20_000 });
  await expect
    .poll(() =>
      responses.some(
        (entry) =>
          entry.method === "GET" &&
          entry.path === "/api/projects" &&
          entry.status === 200
      )
    )
    .toBe(true);
  await expect
    .poll(() =>
      responses.some(
        (entry) =>
          entry.method === "GET" &&
          /\/api\/workspaces\/[^/]+\/members/.test(entry.path) &&
          entry.status === 200
      )
    )
    .toBe(true);
  await expect
    .poll(() =>
      responses.some(
        (entry) =>
          entry.method === "GET" &&
          entry.path === "/api/pins" &&
          entry.status === 200
      )
    )
    .toBe(true);

  const title = `Browser Project ${Date.now()}`;
  await page
    .getByRole("button", { name: /New project|Create your first project/ })
    .first()
    .click();
  await page.getByRole("textbox", { name: "Project title" }).fill(title);
  await page.getByRole("button", { name: "Create Project" }).click();
  await page.waitForURL(new RegExp(`/${SLUG}/projects/[^/]+$`));
  await expect(page.getByText(title, { exact: true }).first()).toBeVisible();
  await expect(page.getByTitle("Pin to sidebar")).toBeVisible();
  await page.getByTitle("Pin to sidebar").click();
  await expect
    .poll(() =>
      responses.some(
        (entry) =>
          entry.method === "POST" &&
          entry.path === "/api/pins" &&
          entry.status === 201
      )
    )
    .toBe(true);
  await expect(page.getByText("API error: 404 Not Found")).toHaveCount(0);

  await page
    .getByTestId("content")
    .getByRole("button", { name: "New Issue" })
    .click();
  const issueTitle = `Project Issue ${Date.now()}`;
  await page.getByRole("textbox", { name: "Issue title" }).fill(issueTitle);
  await page.getByRole("button", { name: "Create Issue" }).click();
  await expect
    .poll(() =>
      responses.some(
        (entry) =>
          entry.method === "POST" &&
          entry.path === "/api/issues" &&
          entry.status === 201
      )
    )
    .toBe(true);
  await expect
    .poll(() =>
      wsFrames.some(
        (frame) => frame.includes("issue:created") && frame.includes(issueTitle)
      )
    )
    .toBe(true);
  await expect(
    page.getByText(issueTitle, { exact: true }).first()
  ).toBeVisible();
  await page.reload({ waitUntil: "domcontentloaded" });
  await expect(page.getByText(issueTitle, { exact: true }).first()).toBeVisible(
    { timeout: 20_000 }
  );
  await expect(page.getByText("API error: 404 Not Found")).toHaveCount(0);

  const projectID = new URL(page.url()).pathname.split("/").at(-1)!;
  const cleanupStatus = await page.evaluate(
    async ({ slug, projectID }) => {
      const csrf = document.cookie
        .split("; ")
        .find((item) => item.startsWith("multica_csrf="))
        ?.split("=")[1];
      const response = await fetch(`/api/projects/${projectID}`, {
        method: "DELETE",
        headers: {
          "X-Workspace-Slug": slug,
          ...(csrf ? { "X-CSRF-Token": decodeURIComponent(csrf) } : {}),
        },
      });
      return response.status;
    },
    { slug: SLUG, projectID }
  );
  expect(cleanupStatus).toBe(204);

  const projectSurface404s = responses.filter(
    (entry) =>
      entry.status === 404 &&
      (entry.path.startsWith("/api/projects") ||
        entry.path === "/api/pins" ||
        entry.path === "/api/issues" ||
        entry.path === "/api/properties" ||
        entry.path === "/api/issues/child-progress" ||
        entry.path === "/api/labels" ||
        /\/api\/workspaces\/[^/]+\/members/.test(entry.path))
  );
  expect(projectSurface404s).toEqual([]);
  const tracePath = testInfo.outputPath(
    "canonical-projects-network-trace.json"
  );
  await writeFile(
    tracePath,
    `${JSON.stringify(
      {
        title,
        issueTitle,
        responses,
        wsEventTypes: wsFrames.map((frame) => {
          try {
            return (JSON.parse(frame) as { type?: string }).type ?? "unknown";
          } catch {
            return "invalid";
          }
        }),
      },
      null,
      2
    )}\n`
  );
  await testInfo.attach("canonical-projects-network-trace", {
    path: tracePath,
    contentType: "application/json",
  });
});

test("C8 clean candidate uploads, previews and downloads a real file", async ({
  page,
}, testInfo) => {
  test.skip(C8_PHASE !== "upload", `C8 phase is ${C8_PHASE ?? "unset"}`);
  const evidenceDir = attachmentEvidenceDir();
  await mkdir(evidenceDir, { recursive: true });
  const content = `C8 clean-candidate attachment ${Date.now()}\n`;
  const filename = "c8-clean-candidate.txt";
  const uploadPath = path.join(evidenceDir, filename);
  await writeFile(uploadPath, content);

  const responses: Array<{ method: string; url: string; status: number }> = [];
  const wsFrames: string[] = [];
  page.on("response", (response) => {
    responses.push({
      method: response.request().method(),
      url: response.url(),
      status: response.status(),
    });
  });
  page.on("websocket", (socket) => {
    socket.on("framereceived", (event) => wsFrames.push(String(event.payload)));
  });

  const workspaceSocket = page.waitForEvent("websocket", {
    predicate: (socket) => new URL(socket.url()).pathname === "/ws",
  });
  await loginFixture(page);
  await openFixtureIssue(page);
  await workspaceSocket;
  const realtimeReadinessAttempts = await awaitCookieRealtimeSubscription(
    page,
    wsFrames
  );
  const uploadResponsePromise = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === "/api/upload-file" &&
      response.request().method() === "POST"
  );
  await uploadFromDescriptionControl(page, uploadPath);
  const uploadResponse = await uploadResponsePromise;
  expect(uploadResponse.status()).toBe(200);
  const attachment = (await uploadResponse.json()) as {
    id: string;
    filename: string;
    size_bytes: number;
  };
  expect(attachment.filename).toBe(filename);
  expect(attachment.size_bytes).toBe(Buffer.byteLength(content));
  await expect(page.getByText(filename, { exact: true }).first()).toBeVisible();

  const persistedRow = page
    .locator(`[data-attachment-id="${attachment.id}"]`)
    .last();
  await expect(persistedRow).toBeVisible();
  const previewButtons = persistedRow.getByRole("button", {
    name: /Preview|预览/,
  });
  await expect(previewButtons).toHaveCount(1);
  const previewResponsePromise = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname ===
      `/api/attachments/${attachment.id}/content`
  );
  await previewButtons.click();
  const previewResponse = await previewResponsePromise;
  expect(previewResponse.status()).toBe(200);
  const dialog = page.getByRole("dialog", { name: filename });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByText(content.trim(), { exact: true })).toBeVisible();

  const downloadPromise = page.waitForEvent("download");
  await dialog.getByRole("button", { name: /Download|下载/ }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toBe(filename);
  await dialog.getByRole("button", { name: /Close|关闭/ }).click();
  await expect(dialog).toBeHidden();

  await expect
    .poll(() =>
      wsFrames.some(
        (frame) =>
          frame.includes("issue_attachments:changed") &&
          frame.includes(attachment.id)
      )
    )
    .toBe(true);
  const listed = await page.request.get(
    `${WEB}/api/issues/${ISSUE_ID}/attachments`,
    { headers: { "X-Workspace-Slug": SLUG } }
  );
  responses.push({ method: "GET", url: listed.url(), status: listed.status() });
  expect(listed.status()).toBe(200);
  const listBody = (await listed.json()) as Array<{
    id: string;
    filename: string;
  }>;
  expect(listBody.some(({ id }) => id === attachment.id)).toBe(true);
  const evidence: AttachmentEvidence = {
    id: attachment.id,
    filename,
    content,
    sizeBytes: Buffer.byteLength(content),
    cleanupIds: [attachment.id],
  };
  await writeFile(
    path.join(evidenceDir, "attachment.json"),
    `${JSON.stringify(evidence, null, 2)}\n`
  );
  const trace = {
    candidate: attachmentCandidate(),
    phase: "upload-preview-download",
    browser: await browserIdentity(page),
    attachment: { id: attachment.id, filename, sizeBytes: evidence.sizeBytes },
    realtimeReadinessAttempts,
    http: sanitizeAttachmentTrace(responses),
    receivedEventTypes: wsFrames
      .map((frame) => {
        try {
          return JSON.parse(frame)?.type;
        } catch {
          return "unparseable";
        }
      })
      .filter((value): value is string => typeof value === "string"),
  };
  const tracePath = path.join(evidenceDir, "upload-trace.json");
  await writeFile(tracePath, `${JSON.stringify(trace, null, 2)}\n`);
  await testInfo.attach("C8 upload preview download trace", {
    path: tracePath,
    contentType: "application/json",
  });
  const screenshotPath = path.join(evidenceDir, "attachment-visible.png");
  await page.screenshot({ path: screenshotPath, fullPage: false });
  await testInfo.attach("C8 attachment visible", {
    path: screenshotPath,
    contentType: "image/png",
  });
});

test("C8 clean candidate reads the same file after runtime restart", async ({
  page,
}, testInfo) => {
  test.skip(C8_PHASE !== "readback", `C8 phase is ${C8_PHASE ?? "unset"}`);
  const evidenceDir = attachmentEvidenceDir();
  const evidence = JSON.parse(
    await readFile(path.join(evidenceDir, "attachment.json"), "utf8")
  ) as AttachmentEvidence;
  const responses: Array<{ method: string; url: string; status: number }> = [];
  page.on("response", (response) => {
    responses.push({
      method: response.request().method(),
      url: response.url(),
      status: response.status(),
    });
  });
  await loginFixture(page);
  await openFixtureIssue(page);

  const metadata = await page.request.get(
    `${WEB}/api/attachments/${evidence.id}`,
    { headers: { "X-Workspace-Slug": SLUG } }
  );
  responses.push({
    method: "GET",
    url: metadata.url(),
    status: metadata.status(),
  });
  expect(metadata.status()).toBe(200);
  const metadataBody = (await metadata.json()) as {
    id: string;
    filename: string;
    size_bytes: number;
  };
  expect(metadataBody).toMatchObject({
    id: evidence.id,
    filename: evidence.filename,
    size_bytes: evidence.sizeBytes,
  });
  const content = await page.request.get(
    `${WEB}/api/attachments/${evidence.id}/content`,
    { headers: { "X-Workspace-Slug": SLUG } }
  );
  responses.push({
    method: "GET",
    url: content.url(),
    status: content.status(),
  });
  expect(content.status()).toBe(200);
  expect(await content.text()).toBe(evidence.content);
  const download = await page.request.get(
    `${WEB}/api/attachments/${evidence.id}/download`,
    { headers: { "X-Workspace-Slug": SLUG } }
  );
  responses.push({
    method: "GET",
    url: download.url(),
    status: download.status(),
  });
  expect(download.status()).toBe(200);
  expect(await download.body()).toEqual(Buffer.from(evidence.content));
  const listed = await page.request.get(
    `${WEB}/api/issues/${ISSUE_ID}/attachments`,
    { headers: { "X-Workspace-Slug": SLUG } }
  );
  responses.push({ method: "GET", url: listed.url(), status: listed.status() });
  expect(listed.status()).toBe(200);
  const listBody = (await listed.json()) as Array<{ id: string }>;
  expect(listBody.some(({ id }) => id === evidence.id)).toBe(true);

  const secondContent = `C8 after-restart attachment ${Date.now()}\n`;
  const secondFilename = "c8-clean-candidate-after-restart.txt";
  const secondUploadPath = path.join(evidenceDir, secondFilename);
  await writeFile(secondUploadPath, secondContent);
  const uploadResponsePromise = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === "/api/upload-file" &&
      response.request().method() === "POST"
  );
  await uploadFromDescriptionControl(page, secondUploadPath);
  const uploadResponse = await uploadResponsePromise;
  expect(uploadResponse.status()).toBe(200);
  const second = (await uploadResponse.json()) as {
    id: string;
    filename: string;
    size_bytes: number;
  };
  expect(second).toMatchObject({
    filename: secondFilename,
    size_bytes: Buffer.byteLength(secondContent),
  });
  await expect(
    page.getByText(secondFilename, { exact: true }).first()
  ).toBeVisible();

  await expect
    .poll(async () => {
      const response = await page.request.get(
        `${WEB}/api/issues/${ISSUE_ID}/attachments`,
        {
          headers: { "X-Workspace-Slug": SLUG },
        }
      );
      if (response.status() !== 200) return [];
      const values = (await response.json()) as Array<{ id: string }>;
      return values
        .map(({ id }) => id)
        .filter((id) => id === evidence.id || id === second.id)
        .sort();
    })
    .toEqual([evidence.id, second.id].sort());
  evidence.afterRestart = {
    id: second.id,
    filename: secondFilename,
    content: secondContent,
    sizeBytes: Buffer.byteLength(secondContent),
  };
  evidence.cleanupIds = [evidence.id, second.id];
  await writeFile(
    path.join(evidenceDir, "attachment.json"),
    `${JSON.stringify(evidence, null, 2)}\n`
  );

  const tracePath = path.join(evidenceDir, "restart-readback-trace.json");
  await writeFile(
    tracePath,
    `${JSON.stringify(
      {
        candidate: attachmentCandidate(),
        phase: "restart-readback",
        browser: await browserIdentity(page),
        attachments: [
          { id: evidence.id, filename: evidence.filename },
          { id: second.id, filename: second.filename },
        ],
        http: sanitizeAttachmentTrace(responses),
      },
      null,
      2
    )}\n`
  );
  await testInfo.attach("C8 restart readback trace", {
    path: tracePath,
    contentType: "application/json",
  });
});

test("C8 clean candidate deletes only the synthetic attachment", async ({
  page,
}, testInfo) => {
  test.skip(C8_PHASE !== "delete", `C8 phase is ${C8_PHASE ?? "unset"}`);
  const evidenceDir = attachmentEvidenceDir();
  const evidence = JSON.parse(
    await readFile(path.join(evidenceDir, "attachment.json"), "utf8")
  ) as AttachmentEvidence;
  const responses: Array<{ method: string; url: string; status: number }> = [];
  page.on("response", (response) => {
    responses.push({
      method: response.request().method(),
      url: response.url(),
      status: response.status(),
    });
  });
  await loginFixture(page);
  await openFixtureIssue(page);
  const keyboardActivations: Array<{ id: string; key: "Enter" | "Space" }> = [];
  for (const [index, attachmentId] of evidence.cleanupIds.entries()) {
    const row = page.locator(`[data-attachment-id="${attachmentId}"]`).last();
    await expect(row).toBeVisible();
    const remove = row.getByRole("button", {
      name: /Remove attachment|删除附件/,
    });
    const key = index === 0 ? "Enter" : "Space";
    const deletedPromise = page.waitForResponse(
      (response) =>
        new URL(response.url()).pathname ===
          `/api/attachments/${attachmentId}` &&
        response.request().method() === "DELETE"
    );
    await remove.focus();
    await expect(remove).toBeFocused();
    await remove.press(key);
    const deleted = await deletedPromise;
    expect(deleted.status()).toBe(204);
    keyboardActivations.push({ id: attachmentId, key });
    await expect(row).toBeHidden();
  }
  const missing = await page.request.get(
    `${WEB}/api/attachments/${evidence.id}`,
    { headers: { "X-Workspace-Slug": SLUG } }
  );
  responses.push({
    method: "GET",
    url: missing.url(),
    status: missing.status(),
  });
  expect(missing.status()).toBe(404);
  const listed = await page.request.get(
    `${WEB}/api/issues/${ISSUE_ID}/attachments`,
    { headers: { "X-Workspace-Slug": SLUG } }
  );
  responses.push({ method: "GET", url: listed.url(), status: listed.status() });
  expect(listed.status()).toBe(200);
  const listBody = (await listed.json()) as Array<{ id: string }>;
  expect(listBody.some(({ id }) => evidence.cleanupIds.includes(id))).toBe(
    false
  );
  const tracePath = path.join(evidenceDir, "delete-trace.json");
  await writeFile(
    tracePath,
    `${JSON.stringify(
      {
        candidate: attachmentCandidate(),
        phase: "delete",
        browser: await browserIdentity(page),
        attachment: { id: evidence.id, filename: evidence.filename },
        keyboard_activations: keyboardActivations,
        http: sanitizeAttachmentTrace(responses),
      },
      null,
      2
    )}\n`
  );
  await testInfo.attach("C8 delete trace", {
    path: tracePath,
    contentType: "application/json",
  });
});

type C9TraceEntry = { method: string; url: string; status: number };

interface C9Artifact {
  candidate: string;
  run: string;
  project: { id: string; title: string };
  pin: { id: string; item_id: string; item_type: string };
  label: { id: string; name: string };
  property: { id: string; name: string; value: string };
  childTitle: string;
  commentText: string;
  acceptanceRationale: string;
  attachment: { id: string; filename: string; content: string };
}

async function c9Checked(
  response: APIResponse,
  trace: C9TraceEntry[],
  method: string,
  accepted: number[] = [200, 201, 204]
) {
  trace.push({ method, url: response.url(), status: response.status() });
  expect(accepted, `${method} ${response.url()}`).toContain(response.status());
  return response;
}

function c9SanitizedTrace(trace: C9TraceEntry[]) {
  return trace.map(({ method, url, status }) => {
    const parsed = new URL(url);
    return { method, origin: parsed.origin, path: parsed.pathname, status };
  });
}

function c9UnexpectedFailures(trace: C9TraceEntry[]) {
  return c9SanitizedTrace(trace).filter(({ path: requestPath, status }) => {
    if (requestPath === "/api/invitations" && status === 404) return false;
    return status >= 400;
  });
}

async function c9CSRFHeaders(page: Page) {
  const cookie = (await page.context().cookies(WEB)).find(
    (value) => value.name === "multica_csrf"
  );
  expect(cookie).toBeDefined();
  return {
    "X-Workspace-Slug": SLUG,
    "X-CSRF-Token": cookie!.value,
  };
}

async function expectC9VisibleText(page: Page, value: string) {
  const matches = page.getByText(value, { exact: true });
  await expect
    .poll(() => matches.count(), { timeout: 30_000 })
    .toBeGreaterThan(0);
  await expect(matches.first()).toBeVisible();
}

test("C9 clean candidate exercises complete local Issue detail before restart", async ({
  page,
}, testInfo) => {
  test.skip(C9_PHASE !== "pre", `C9 phase is ${C9_PHASE ?? "unset"}`);
  const evidenceDir = c9EvidenceDir();
  await mkdir(evidenceDir, { recursive: true });
  const run = `${Date.now()}`;
  const trace: C9TraceEntry[] = [];
  const wsFrames: string[] = [];
  const websockets: Array<{ origin: string; path: string }> = [];
  page.on("response", (response) => {
    if (new URL(response.url()).origin === WEB) {
      trace.push({
        method: response.request().method(),
        url: response.url(),
        status: response.status(),
      });
    }
  });
  page.on("websocket", (socket) => {
    const parsed = new URL(socket.url());
    websockets.push({ origin: parsed.origin, path: parsed.pathname });
    socket.on("framereceived", (event) => wsFrames.push(String(event.payload)));
  });

  const socketReady = page.waitForEvent("websocket", {
    predicate: (socket) => new URL(socket.url()).pathname === "/ws",
  });
  await loginFixture(page);
  await socketReady;
  await openFixtureIssue(page);
  await expectC9VisibleText(page, "Canonical runtime acceptance");
  await expect(
    page.getByRole("button", { name: "Attach file" }).first()
  ).toBeVisible();
  const headers = await c9CSRFHeaders(page);
  // Authentication bootstrap may intentionally probe protected endpoints before
  // the cookie session is established. The C9 route gate starts at the
  // authenticated detail interaction, where every failure is actionable.
  trace.length = 0;

  const projectResponse = await c9Checked(
    await page.request.post(`${WEB}/api/projects`, {
      headers,
      data: {
        title: `C9 Project ${run}`,
        description: "C9 complete-detail acceptance",
        status: "in_progress",
        priority: "high",
      },
    }),
    trace,
    "POST"
  );
  const project = (await projectResponse.json()) as {
    id: string;
    title: string;
  };
  const pinResponse = await c9Checked(
    await page.request.post(`${WEB}/api/pins`, {
      headers,
      data: { item_type: "project", item_id: project.id },
    }),
    trace,
    "POST"
  );
  const pin = (await pinResponse.json()) as {
    id: string;
    item_id: string;
    item_type: string;
  };
  expect(pin).toMatchObject({ item_id: project.id, item_type: "project" });

  const labelResponse = await c9Checked(
    await page.request.post(`${WEB}/api/labels`, {
      headers,
      data: {
        resource_type: "issue",
        name: `c9-label-${run}`,
        description: "C9 acceptance label",
        color: "#3b82f6",
      },
    }),
    trace,
    "POST"
  );
  const label = (await labelResponse.json()) as { id: string; name: string };
  const propertyResponse = await c9Checked(
    await page.request.post(`${WEB}/api/properties`, {
      headers,
      data: {
        name: `C9 Property ${run}`,
        type: "text",
        description: "C9 acceptance property",
      },
    }),
    trace,
    "POST"
  );
  const property = (await propertyResponse.json()) as {
    id: string;
    name: string;
  };
  const propertyValue = `verified-${run}`;
  const description = `C9 complete detail ${run}`;

  await c9Checked(
    await page.request.put(`${WEB}/api/issues/${ISSUE_ID}`, {
      headers,
      data: {
        description,
        priority: "high",
        project_id: project.id,
        start_date: "2026-08-15",
        due_date: "2026-08-20",
      },
    }),
    trace,
    "PUT"
  );
  await c9Checked(
    await page.request.post(`${WEB}/api/issues/${ISSUE_ID}/move`, {
      headers,
      data: { status: "done", before_id: null, after_id: null },
    }),
    trace,
    "POST"
  );

  const childTitle = `C9 child ${run}`;
  await c9Checked(
    await page.request.post(`${WEB}/api/issues`, {
      headers,
      data: {
        title: childTitle,
        status: "todo",
        priority: "medium",
        parent_issue_id: ISSUE_ID,
        project_id: project.id,
        stage: 1,
      },
    }),
    trace,
    "POST"
  );
  const commentText = `C9 collaboration ${run}`;
  await c9Checked(
    await page.request.post(`${WEB}/api/issues/${ISSUE_ID}/comments`, {
      headers,
      data: { content: commentText, type: "comment" },
    }),
    trace,
    "POST"
  );
  await c9Checked(
    await page.request.post(`${WEB}/api/issues/${ISSUE_ID}/subscribe`, {
      headers,
      data: {},
    }),
    trace,
    "POST"
  );
  await c9Checked(
    await page.request.post(`${WEB}/api/issues/${ISSUE_ID}/reactions`, {
      headers,
      data: { emoji: "✅" },
    }),
    trace,
    "POST"
  );
  await c9Checked(
    await page.request.post(`${WEB}/api/issues/${ISSUE_ID}/labels`, {
      headers,
      data: { label_id: label.id },
    }),
    trace,
    "POST"
  );
  await c9Checked(
    await page.request.put(
      `${WEB}/api/issues/${ISSUE_ID}/properties/${property.id}`,
      { headers, data: { value: propertyValue } }
    ),
    trace,
    "PUT"
  );
  const acceptanceRationale = `C9 acceptance ${run}`;
  await c9Checked(
    await page.request.post(
      `${WEB}/api/issues/${ISSUE_ID}/acceptance-conclusions`,
      {
        headers,
        data: {
          result: "accepted",
          rationale: acceptanceRationale,
          evidence_refs: [`local://c9/${run}`],
        },
      }
    ),
    trace,
    "POST"
  );

  // Reload before the visible upload so the editor starts from the complete
  // authoritative Issue/attachment bag written by the API-assisted setup.
  await page.reload({ waitUntil: "networkidle" });
  await expectC9VisibleText(page, description);

  const attachmentFilename = `c9-attachment-${run}.txt`;
  const attachmentContent = `C9 attachment ${run}\n`;
  const attachmentPath = path.join(evidenceDir, attachmentFilename);
  await writeFile(attachmentPath, attachmentContent);
  const uploadPromise = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === "/api/upload-file" &&
      response.request().method() === "POST"
  );
  await uploadFromDescriptionControl(page, attachmentPath);
  const uploaded = await uploadPromise;
  trace.push({
    method: "POST",
    url: uploaded.url(),
    status: uploaded.status(),
  });
  expect(uploaded.status()).toBe(200);
  const attachment = (await uploaded.json()) as {
    id: string;
    filename: string;
  };
  await expectC9VisibleText(page, attachment.filename);

  await expect
    .poll(
      () =>
        [
          "issue:updated",
          "issue:created",
          "comment:created",
          "subscriber:added",
          "issue_reaction:added",
          "issue_labels:changed",
          "issue_properties:changed",
          "issue_attachments:changed",
        ].every((type) => wsFrames.some((frame) => frame.includes(type))),
      { timeout: 20_000 }
    )
    .toBe(true);

  await page.reload({ waitUntil: "networkidle" });
  await expectC9VisibleText(page, description);
  await expectC9VisibleText(page, project.title);
  await expectC9VisibleText(page, label.name);
  await expectC9VisibleText(page, property.name);
  await expectC9VisibleText(page, propertyValue);
  await expectC9VisibleText(page, attachment.filename);

  const readbacks = await Promise.all([
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE}`, { headers }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/timeline`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/subscribers`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/reactions`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/children`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/labels`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(
        `${WEB}/api/issues/${ISSUE_ID}/acceptance-conclusions`,
        { headers }
      ),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/attachments`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/pins`, { headers }),
      trace,
      "GET"
    ),
  ]);
  const bodies = await Promise.all(
    readbacks.map((response) => response.text())
  );
  expect(bodies[0]).toContain(project.id);
  expect(bodies[0]).toContain(propertyValue);
  expect(bodies[1]).toContain(commentText);
  expect(bodies[3]).toContain("✅");
  expect(bodies[4]).toContain(childTitle);
  expect(bodies[5]).toContain(label.id);
  expect(bodies[6]).toContain(acceptanceRationale);
  expect(bodies[7]).toContain(attachment.id);
  expect(bodies[8]).toContain(pin.id);
  expect(c9UnexpectedFailures(trace)).toEqual([]);
  expect(c9SanitizedTrace(trace).every(({ origin }) => origin === WEB)).toBe(
    true
  );
  expect(websockets).toContainEqual({
    origin: "ws://127.0.0.1:3000",
    path: "/ws",
  });
  expect(
    [
      ...c9SanitizedTrace(trace).map(({ origin }) => origin),
      ...websockets.map(({ origin }) => origin),
    ].some((origin) => new URL(origin).port === "8080")
  ).toBe(false);

  const artifact: C9Artifact = {
    candidate: c9Candidate(),
    run,
    project,
    pin,
    label,
    property: { ...property, value: propertyValue },
    childTitle,
    commentText,
    acceptanceRationale,
    attachment: {
      id: attachment.id,
      filename: attachment.filename,
      content: attachmentContent,
    },
  };
  const tracePath = path.join(evidenceDir, "c9-pre-restart.json");
  await writeFile(
    tracePath,
    `${JSON.stringify(
      {
        ...artifact,
        browser: await browserIdentity(page),
        http: c9SanitizedTrace(trace),
        websockets,
        receivedEventTypes: [
          ...new Set(
            wsFrames.map((frame) => {
              try {
                return (
                  (JSON.parse(frame) as { type?: string }).type ?? "unknown"
                );
              } catch {
                return "invalid";
              }
            })
          ),
        ].sort(),
      },
      null,
      2
    )}\n`
  );
  const screenshotPath = path.join(evidenceDir, "c9-pre-restart.png");
  await page.screenshot({ path: screenshotPath, fullPage: true });
  await testInfo.attach("C9 complete detail pre-restart", {
    path: tracePath,
    contentType: "application/json",
  });
  await testInfo.attach("C9 complete detail pre-restart screenshot", {
    path: screenshotPath,
    contentType: "image/png",
  });
});

test("C9 clean candidate retains complete local Issue detail after restart", async ({
  page,
}, testInfo) => {
  test.skip(C9_PHASE !== "post", `C9 phase is ${C9_PHASE ?? "unset"}`);
  const evidenceDir = c9EvidenceDir();
  const artifact = JSON.parse(
    await readFile(path.join(evidenceDir, "c9-pre-restart.json"), "utf8")
  ) as C9Artifact;
  const trace: C9TraceEntry[] = [];
  const websockets: Array<{ origin: string; path: string }> = [];
  page.on("response", (response) => {
    if (new URL(response.url()).origin === WEB) {
      trace.push({
        method: response.request().method(),
        url: response.url(),
        status: response.status(),
      });
    }
  });
  page.on("websocket", (socket) => {
    const parsed = new URL(socket.url());
    websockets.push({ origin: parsed.origin, path: parsed.pathname });
  });

  await loginFixture(page);
  await openFixtureIssue(page);
  const headers = await c9CSRFHeaders(page);
  trace.length = 0;
  await expectC9VisibleText(page, `C9 complete detail ${artifact.run}`);
  await expectC9VisibleText(page, artifact.project.title);
  await expectC9VisibleText(page, artifact.label.name);
  await expectC9VisibleText(page, artifact.property.name);
  await expectC9VisibleText(page, artifact.property.value);
  await expectC9VisibleText(page, artifact.attachment.filename);

  const readbacks = await Promise.all([
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE}`, { headers }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/timeline`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/subscribers`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/reactions`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/children`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/labels`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(
        `${WEB}/api/issues/${ISSUE_ID}/acceptance-conclusions`,
        { headers }
      ),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/issues/${ISSUE_ID}/attachments`, {
        headers,
      }),
      trace,
      "GET"
    ),
    c9Checked(
      await page.request.get(`${WEB}/api/pins`, { headers }),
      trace,
      "GET"
    ),
  ]);
  const bodies = await Promise.all(
    readbacks.map((response) => response.text())
  );
  expect(bodies[0]).toContain(artifact.project.id);
  expect(bodies[0]).toContain(artifact.property.value);
  expect(bodies[1]).toContain(artifact.commentText);
  expect(bodies[3]).toContain("✅");
  expect(bodies[4]).toContain(artifact.childTitle);
  expect(bodies[5]).toContain(artifact.label.id);
  expect(bodies[6]).toContain(artifact.acceptanceRationale);
  expect(bodies[7]).toContain(artifact.attachment.id);
  expect(bodies[8]).toContain(artifact.pin.id);
  expect(c9UnexpectedFailures(trace)).toEqual([]);
  expect(c9SanitizedTrace(trace).every(({ origin }) => origin === WEB)).toBe(
    true
  );
  await expect
    .poll(() =>
      websockets.some(({ path: requestPath }) => requestPath === "/ws")
    )
    .toBe(true);

  const tracePath = path.join(evidenceDir, "c9-post-restart.json");
  await writeFile(
    tracePath,
    `${JSON.stringify(
      {
        candidate: c9Candidate(),
        run: artifact.run,
        browser: await browserIdentity(page),
        retained: {
          project: artifact.project.id,
          pin: artifact.pin.id,
          label: artifact.label.id,
          property: artifact.property.id,
          attachment: artifact.attachment.id,
        },
        http: c9SanitizedTrace(trace),
        websockets,
      },
      null,
      2
    )}\n`
  );
  const screenshotPath = path.join(evidenceDir, "c9-post-restart.png");
  await page.screenshot({ path: screenshotPath, fullPage: true });
  await testInfo.attach("C9 complete detail post-restart", {
    path: tracePath,
    contentType: "application/json",
  });
  await testInfo.attach("C9 complete detail post-restart screenshot", {
    path: screenshotPath,
    contentType: "image/png",
  });
});
