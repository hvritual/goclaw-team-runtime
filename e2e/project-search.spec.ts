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

async function createProject(request: APIRequestContext, token: string, slug: string, title: string, description?: string) {
  const response = await request.post("/api/projects", {
    headers: { Authorization: `Bearer ${token}`, "X-Workspace-Slug": slug },
    data: { title, ...(description ? { description } : {}) },
  });
  expect(response.status(), await response.text()).toBe(201);
  return (await response.json()) as { id: string; title: string };
}

async function searchProjects(page: Page, token: string, query: string, suffix = "") {
  const response = await page.request.get(`/api/projects/search?q=${encodeURIComponent(query)}${suffix}`, {
    headers: { Authorization: `Bearer ${token}`, "X-Workspace-Slug": SLUG },
  });
  expect(response.status(), await response.text()).toBe(200);
  return (await response.json()) as { projects: Array<{ id: string; title: string; match_source: "title" | "description" }>; total: number };
}

test("installed Project search handles Unicode, closed state, pagination, isolation, and shared UI", async ({ page }) => {
  const { token } = await loginFixture(page);
  await page.context().clearCookies();
  await page.context().addCookies([{ name: "multica-locale", value: "en", url: "http://127.0.0.1:3000" }]);
  const marker = Date.now().toString(36);
  const englishTitle = `Alpha Beta project ${marker}`;
  const chineseTitle = `修复咖啡机项目 ${marker}`;
  const closedTitle = `Alpha Beta closed project ${marker}`;
  const english = await createProject(page.request, token, SLUG, englishTitle);
  const chinese = await createProject(page.request, token, SLUG, chineseTitle);
  const description = await createProject(page.request, token, SLUG, `Other project ${marker}`, `Alpha Beta description ${marker}`);
  const closed = await createProject(page.request, token, SLUG, closedTitle);
  const closeResponse = await page.request.put(`/api/projects/${closed.id}`, {
    headers: { Authorization: `Bearer ${token}`, "X-Workspace-Slug": SLUG },
    data: { status: "completed" },
  });
  expect(closeResponse.status(), await closeResponse.text()).toBe(200);

  const openOnly = await searchProjects(page, token, `alpha beta ${marker}`);
  expect(openOnly.projects.map((project) => project.id)).toEqual([english.id, description.id]);
  expect(openOnly.projects.map((project) => project.match_source)).toEqual(["title", "description"]);
  const firstPage = await searchProjects(page, token, `alpha beta ${marker}`, "&include_closed=true&limit=1");
  const secondPage = await searchProjects(page, token, `alpha beta ${marker}`, "&include_closed=true&limit=1&offset=1");
  const thirdPage = await searchProjects(page, token, `alpha beta ${marker}`, "&include_closed=true&limit=1&offset=2");
  expect(firstPage.total).toBe(3);
  expect(new Set([firstPage.projects[0]?.id, secondPage.projects[0]?.id, thirdPage.projects[0]?.id])).toEqual(
    new Set([english.id, closed.id, description.id]),
  );
  expect((await searchProjects(page, token, `咖啡机 ${marker}`)).projects[0]?.id).toBe(chinese.id);

  const otherSlug = `project-search-${marker}`;
  const otherWorkspace = await page.request.post("/api/workspaces", {
    headers: { Authorization: `Bearer ${token}` },
    data: { name: `Other Project search ${marker}`, slug: otherSlug },
  });
  expect(otherWorkspace.status(), await otherWorkspace.text()).toBe(201);
  const foreign = await createProject(page.request, token, otherSlug, `Alpha Beta ${marker} foreign`);
  const isolated = await searchProjects(page, token, `alpha beta ${marker}`, "&include_closed=true&limit=50");
  expect(isolated.projects.some((project) => project.id === foreign.id)).toBe(false);

  await loginBrowserFixture(page);
  await page.getByRole("button", { name: "Search..." }).click();
  const input = page.getByPlaceholder("Type a command or search...");
  await input.fill(`咖啡机 ${marker}`);
  await expect(page.getByText(chineseTitle, { exact: true })).toBeVisible();
  await input.fill(`alpha beta ${marker}`);
  await expect(page.getByText(englishTitle, { exact: true })).toBeVisible();
  await expect(page.getByText(closedTitle, { exact: true })).toBeVisible();
  await expect(page.getByText(`Other project ${marker}`, { exact: true })).toBeVisible();
});
