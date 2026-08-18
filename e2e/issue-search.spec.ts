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

async function createIssue(request: APIRequestContext, token: string, slug: string, title: string) {
  const response = await request.post("/api/issues", {
    headers: { Authorization: `Bearer ${token}`, "X-Workspace-Slug": slug },
    data: { title },
  });
  expect(response.status(), await response.text()).toBe(201);
  return (await response.json()) as { id: string; identifier: string; number: number };
}

async function searchIssues(page: Page, token: string, query: string, suffix = "") {
  const response = await page.request.get(`/api/issues/search?q=${encodeURIComponent(query)}${suffix}`, {
    headers: { Authorization: `Bearer ${token}`, "X-Workspace-Slug": SLUG },
  });
  expect(response.status(), await response.text()).toBe(200);
  return (await response.json()) as { issues: Array<{ id: string; title: string }>; total: number };
}

test("installed Issue search handles ranking, Unicode, closed state, pagination, and isolation", async ({ page }) => {
  const { token } = await loginFixture(page);
  await page.addInitScript((value) => localStorage.setItem("multica_token", value), token);
  const marker = Date.now().toString(36);
  const englishTitle = `Alpha Beta browser ${marker}`;
  const chineseTitle = `修复咖啡机搜索 ${marker}`;
  const closedTitle = `Alpha Beta closed ${marker}`;
  const english = await createIssue(page.request, token, SLUG, englishTitle);
  const chinese = await createIssue(page.request, token, SLUG, chineseTitle);
  const closed = await createIssue(page.request, token, SLUG, closedTitle);
  const closeResponse = await page.request.put(`/api/issues/${closed.id}`, {
    headers: { Authorization: `Bearer ${token}`, "X-Workspace-Slug": SLUG },
    data: { status: "done" },
  });
  expect(closeResponse.status(), await closeResponse.text()).toBe(200);

  const openOnly = await searchIssues(page, token, `alpha beta ${marker}`);
  expect(openOnly.issues.map((issue) => issue.id)).toEqual([english.id]);
  const withClosed = await searchIssues(page, token, `alpha beta ${marker}`, "&include_closed=true&limit=1");
  expect(withClosed.total).toBe(2);
  expect(withClosed.issues).toHaveLength(1);
  const secondPage = await searchIssues(page, token, `alpha beta ${marker}`, "&include_closed=true&limit=1&offset=1");
  expect(secondPage.total).toBe(2);
  expect(secondPage.issues).toHaveLength(1);
  expect(new Set([withClosed.issues[0]?.id, secondPage.issues[0]?.id])).toEqual(new Set([english.id, closed.id]));
  expect((await searchIssues(page, token, english.identifier)).issues[0]?.id).toBe(english.id);
  expect((await searchIssues(page, token, String(english.number))).issues[0]?.id).toBe(english.id);
  expect((await searchIssues(page, token, `咖啡机 ${marker}`)).issues[0]?.id).toBe(chinese.id);

  const otherWorkspace = await page.request.post("/api/workspaces", {
    headers: { Authorization: `Bearer ${token}` },
    data: { name: `Other search ${marker}`, slug: `other-search-${marker}` },
  });
  expect(otherWorkspace.status(), await otherWorkspace.text()).toBe(201);
  const foreign = await createIssue(page.request, token, `other-search-${marker}`, `Alpha Beta ${marker} foreign`);
  const isolated = await searchIssues(page, token, `alpha beta ${marker}`, "&include_closed=true&limit=50");
  expect(isolated.issues.some((issue) => issue.id === foreign.id)).toBe(false);

  await page.goto(`/${SLUG}/issues`, { waitUntil: "domcontentloaded" });
  await page.getByText("Search", { exact: true }).first().click();
  const input = page.getByPlaceholder("Type a command or search...");
  await input.fill(`咖啡机 ${marker}`);
  await expect(page.getByText(chineseTitle, { exact: true })).toBeVisible();
  await input.fill(`alpha beta ${marker}`);
  await expect(page.getByText(englishTitle, { exact: true })).toBeVisible();
  await expect(page.getByText(closedTitle, { exact: true })).toBeVisible();
});
