import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

const EMAIL = "canonical-fixture@multica.local";
const SLUG = "canonical-fixture";

test.setTimeout(120_000);

async function loginFixture(page: Page) {
  const response = await page.request.post("/auth/verify-code", {
    data: { email: EMAIL, code: "888888" },
  });
  expect(response.status()).toBe(200);
  return (await response.json()) as { token: string };
}

async function loginBrowserFixture(page: Page) {
  await page.goto("/login", { waitUntil: "domcontentloaded" });
  await page.locator("#login-email").fill(EMAIL);
  await page.getByRole("button", { name: "Continue" }).click();
  await page.locator('input[autocomplete="one-time-code"]').fill("888888");
  await page.waitForURL(new RegExp(`/${SLUG}/issues`));
}

function headers(token: string) {
  return { Authorization: `Bearer ${token}`, "X-Workspace-Slug": SLUG };
}

async function createProject(request: APIRequestContext, token: string, title: string) {
  const response = await request.post("/api/projects", {
    headers: headers(token),
    data: { title },
  });
  expect(response.status(), await response.text()).toBe(201);
  return (await response.json()) as { id: string };
}

async function createIssue(request: APIRequestContext, token: string, title: string) {
  const response = await request.post("/api/issues", {
    headers: headers(token),
    data: { title },
  });
  expect(response.status(), await response.text()).toBe(201);
  return (await response.json()) as { id: string };
}

async function createPin(request: APIRequestContext, token: string, item_type: "issue" | "project", item_id: string) {
  const response = await request.post("/api/pins", {
    headers: headers(token),
    data: { item_type, item_id },
  });
  expect(response.status(), await response.text()).toBe(201);
  return (await response.json()) as { id: string; order_revision: number };
}

async function expectPinnedOrder(page: Page, hrefs: string[]) {
  const selector = hrefs.map((href) => `a[href="${href}"]`).join(",");
  await expect.poll(async () => page.locator(selector).evaluateAll((nodes) => nodes.map((node) => node.getAttribute("href")))).toEqual(hrefs);
}

test("installed Pin reorder rolls back stale drag and persists one atomic complete order", async ({ page }) => {
  const { token } = await loginFixture(page);
  const marker = Date.now().toString(36);
  const issueTitle = `Pinned issue ${marker}`;
  const projectTitle = `Pinned project ${marker}`;
  const thirdTitle = `Pinned third project ${marker}`;
  const issue = await createIssue(page.request, token, issueTitle);
  const project = await createProject(page.request, token, projectTitle);
  const issuePin = await createPin(page.request, token, "issue", issue.id);
  const projectPin = await createPin(page.request, token, "project", project.id);
  expect([issuePin.order_revision, projectPin.order_revision]).toEqual([1, 2]);

  await page.context().clearCookies();
  await page.context().addCookies([{ name: "multica-locale", value: "en", url: "http://127.0.0.1:3000" }]);
  await loginBrowserFixture(page);
  const issueHref = `/${SLUG}/issues/${issue.id}`;
  const projectHref = `/${SLUG}/projects/${project.id}`;
  await expect(page.locator(`a[href="${issueHref}"]`)).toBeVisible();
  await expect(page.locator(`a[href="${projectHref}"]`)).toBeVisible();
  await expectPinnedOrder(page, [issueHref, projectHref]);

  const third = await createProject(page.request, token, thirdTitle);
  const thirdPin = await createPin(page.request, token, "project", third.id);
  expect(thirdPin.order_revision).toBe(3);

  const staleResponsePromise = page.waitForResponse((response) => response.url().endsWith("/api/pins/reorder") && response.request().method() === "PUT");
  await page.locator(`a[href="${projectHref}"]`).dragTo(page.locator(`a[href="${issueHref}"]`));
  const staleResponse = await staleResponsePromise;
  expect(staleResponse.status()).toBe(409);
  expect(await staleResponse.json()).toEqual({
    code: "revision_conflict",
    current_revision: 3,
    error: "revision conflict",
  });

  const thirdHref = `/${SLUG}/projects/${third.id}`;
  await expect(page.locator(`a[href="${thirdHref}"]`)).toBeVisible();
  await expectPinnedOrder(page, [issueHref, projectHref, thirdHref]);

  const successResponsePromise = page.waitForResponse((response) => response.url().endsWith("/api/pins/reorder") && response.request().method() === "PUT");
  await page.locator(`a[href="${projectHref}"]`).dragTo(page.locator(`a[href="${issueHref}"]`));
  const successResponse = await successResponsePromise;
  expect(successResponse.status()).toBe(204);
  await expectPinnedOrder(page, [projectHref, issueHref, thirdHref]);

  const listed = await page.request.get("/api/pins", {
    headers: headers(token),
  });
  expect(listed.status(), await listed.text()).toBe(200);
  const pins = (await listed.json()) as Array<{
    id: string;
    position: number;
    order_revision: number;
  }>;
  expect(pins.map((pin) => pin.id)).toEqual([projectPin.id, issuePin.id, thirdPin.id]);
  expect(pins.map((pin) => pin.position)).toEqual([1, 2, 3]);
  expect(new Set(pins.map((pin) => pin.order_revision))).toEqual(new Set([4]));

  await page.reload({ waitUntil: "domcontentloaded" });
  await expectPinnedOrder(page, [projectHref, issueHref, thirdHref]);
});
