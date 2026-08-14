import { expect, test } from "@playwright/test";
import { writeFile } from "node:fs/promises";

const WEB = "http://127.0.0.1:3000";
const EMAIL = "canonical-fixture@multica.local";
const SLUG = "canonical-fixture";
const ISSUE = "CAN-1";

test("Canonical new user creates the first Workspace and completes onboarding", async ({ page }, testInfo) => {
  const runId = Date.now();
  const email = `onboarding-${runId}@example.com`;
  const workspaceName = `Onboarding ${runId}`;
  const slug = `onboarding-${runId}`;
  const responses: Array<{ method: string; path: string; status: number }> = [];
  page.on("response", (response) => {
    const request = response.request();
    const url = new URL(response.url());
    responses.push({ method: request.method(), path: url.pathname, status: response.status() });
  });

  await page.goto(`${WEB}/login`, { waitUntil: "domcontentloaded" });
  await page.waitForLoadState("networkidle");
  await page.locator("#login-email").fill(email);
  await page.getByRole("button", { name: "Continue" }).click();
  await page.locator('input[autocomplete="one-time-code"]').fill("888888");
  await page.waitForURL(`${WEB}/onboarding`);

  await page.locator("#ws-name").fill(workspaceName);
  await expect(page.locator("#ws-slug")).toHaveValue(slug);
  const createButton = page.getByRole("button", { name: `Create ${workspaceName}` });
  await expect(createButton).toBeEnabled();
  await createButton.click();
  await page.waitForURL(new RegExp(`/${slug}/issues`));
  await page.reload({ waitUntil: "domcontentloaded" });
  await expect(page).toHaveURL(new RegExp(`/${slug}/issues`));

  expect(responses).toContainEqual({ method: "POST", path: "/api/workspaces", status: 201 });
  expect(responses).toContainEqual({ method: "POST", path: "/api/me/onboarding/complete", status: 200 });
  const tracePath = testInfo.outputPath("canonical-onboarding-trace.json");
  await writeFile(tracePath, `${JSON.stringify({ email, slug, responses }, null, 2)}\n`);
  await testInfo.attach("canonical-onboarding-trace", { path: tracePath, contentType: "application/json" });
});

test("Canonical UI login, Workspace, Issue and metadata journey has no legacy traffic", async ({ page }, testInfo) => {
  const httpTrace: Array<{ method: string; url: string }> = [];
  const wsTrace: string[] = [];
  const wsFrames: string[] = [];
  page.on("request", (request) => httpTrace.push({ method: request.method(), url: request.url() }));
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

  await expect.poll(() => httpTrace.some(({ url }) => url.includes("/api/workspaces"))).toBe(true);
  await page.goto(`${WEB}/${SLUG}/issues`, { waitUntil: "domcontentloaded" });
  await page.getByText("Canonical runtime acceptance", { exact: true }).click();
  await page.waitForURL(new RegExp(`/${SLUG}/issues/(?:${ISSUE}|0199)`));
  await expect(page.getByText("Canonical runtime acceptance")).toBeVisible();

  const value = `browser-${Date.now()}`;
  const metadataTrace = await page.evaluate(async ({ issue, value }) => {
    const csrf = document.cookie.split("; ").find((item) => item.startsWith("multica_csrf="))?.split("=")[1];
    const headers = { "Content-Type": "application/json", "X-Workspace-Slug": "canonical-fixture", ...(csrf ? { "X-CSRF-Token": decodeURIComponent(csrf) } : {}) };
    const urls: string[] = [];
    const call = async (method: string, suffix = "", body?: string) => {
      const url = `/api/issues/${issue}/metadata${suffix}`;
      urls.push(`${method} ${url}`);
      const response = await fetch(url, { method, headers, body });
      if (!response.ok) throw new Error(`${method} ${url}: ${response.status} ${await response.text()}`);
      return response.json();
    };
    await call("GET");
    const put = await call("PUT", "/browser_readback", JSON.stringify({ value }));
    if (put.metadata.browser_readback !== value) throw new Error("PUT readback mismatch");
    const get = await call("GET");
    if (get.metadata.browser_readback !== value) throw new Error("GET readback mismatch");
    return urls;
  }, { issue: ISSUE, value });
  expect(metadataTrace).toEqual([
    `GET /api/issues/${ISSUE}/metadata`,
    `PUT /api/issues/${ISSUE}/metadata/browser_readback`,
    `GET /api/issues/${ISSUE}/metadata`,
  ]);
  await expect.poll(() => wsFrames.some((frame) => frame.includes("issue_metadata:changed") && frame.includes(value))).toBe(true);
  await expect(page.getByTestId("issue-base-detail")).toBeVisible();
  await expect(page.getByText("Canonical runtime acceptance")).toBeVisible();
  const removed = await page.evaluate(async (issue) => {
    const csrf = document.cookie.split("; ").find((item) => item.startsWith("multica_csrf="))?.split("=")[1];
    const headers = { "X-Workspace-Slug": "canonical-fixture", ...(csrf ? { "X-CSRF-Token": decodeURIComponent(csrf) } : {}) };
    const deleted = await fetch(`/api/issues/${issue}/metadata/browser_readback`, { method: "DELETE", headers });
    if (!deleted.ok) throw new Error(`DELETE failed ${deleted.status}`);
    const response = await fetch(`/api/issues/${issue}/metadata`, { headers });
    return response.json();
  }, ISSUE);
  expect(Object.hasOwn(removed.metadata, "browser_readback")).toBe(false);

  expect(httpTrace.some(({ url }) => url.includes("/auth/send-code"))).toBe(true);
  expect(httpTrace.some(({ url }) => url.includes("/auth/verify-code"))).toBe(true);
  expect(httpTrace.some(({ url }) => url.includes(`/api/issues/${ISSUE}`))).toBe(true);
  expect([...httpTrace.map(({ url }) => url), ...wsTrace].every((url) => new URL(url).port !== "8080")).toBe(true);
  expect(wsTrace.some((url) => new URL(url).port === "3000" || new URL(url).port === "8000")).toBe(true);
  await expect.poll(() => wsFrames.filter((frame) => frame.includes("issue_metadata:changed")).length).toBeGreaterThanOrEqual(2);

  const receivedEventTypes = wsFrames.map((frame) => {
    try { return JSON.parse(frame)?.type; } catch { return "unparseable"; }
  }).filter((type): type is string => typeof type === "string");
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
  await testInfo.attach("canonical-network-trace", { path: tracePath, contentType: "application/json" });
});
