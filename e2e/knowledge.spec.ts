import { expect, test, type Page } from "@playwright/test";

const EMAIL = `knowledge-query-${Date.now()}@multica.local`;
const SLUG = `knowledge-query-${Date.now().toString(36)}`;

test.setTimeout(90_000);

async function installOwnerSession(page: Page) {
  const login = await page.request.post("/auth/verify-code", { data: { email: EMAIL, code: "888888" } });
  expect(login.status(), await login.text()).toBe(200);
  const { token } = (await login.json()) as { token: string };
  const headers = { Authorization: `Bearer ${token}` };
  const created = await page.request.post("/api/workspaces", { headers, data: { name: "Knowledge Query", slug: SLUG } });
  expect(created.status(), await created.text()).toBe(201);
  const workspace = (await created.json()) as { id: string };
  await page.request.post("/api/me/onboarding/complete", { headers, data: { completion_path: "full", workspace_id: workspace.id } });
  await page.addInitScript((value) => { localStorage.setItem("multica_token", value); localStorage.setItem("multica:chat:isOpen", "false"); }, token);
  return workspace.id;
}

test("installed Knowledge query renders provenance and keeps review controls absent", async ({ page }) => {
  const workspaceID = await installOwnerSession(page);
  let listURL = "";
  const revision = {
    number: 1, supersedes_revision: 0, title: "Retry safely", content: "Retain evidence while delivery recovers.",
    created_by: "owner", created_at: "2026-08-18T00:00:00Z",
    source_refs: [{ type: "acceptance_conclusion", id: "issue-1", revision: "sha256:abc", citation: "Acceptance passed", asset_id: null, asset_version_id: null }],
  };
  await page.route("**/api/knowledge**", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname.endsWith("/knowledge-1")) {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({
        id: "knowledge-1", workspace_id: workspaceID, project_id: null, candidate_id: "candidate-1", kind: "lesson", status: "published", current_revision: 1,
        revision, revisions: [revision], citation: "Acceptance passed", matched_by: "detail", created_at: "2026-08-18T00:00:00Z", updated_at: "2026-08-18T00:00:00Z",
      }) });
      return;
    }
    listURL = route.request().url();
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ entries: [{
      id: "knowledge-1", workspace_id: workspaceID, project_id: null, candidate_id: "candidate-1", kind: "lesson", status: "published", current_revision: 1,
      revision, citation: "Acceptance passed", matched_by: "source", created_at: "2026-08-18T00:00:00Z", updated_at: "2026-08-18T00:00:00Z",
    }], total: 1, next_cursor: null }) });
  });

  await page.goto(`/${SLUG}/knowledge?source_type=acceptance_conclusion&source_id=issue-1&source_revision=sha256%3Aabc`, { waitUntil: "domcontentloaded" });
  await expect(page.getByText("Retry safely", { exact: true })).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText("Acceptance passed", { exact: true })).toBeVisible();
  expect(listURL).toContain("source_type=acceptance_conclusion");
  expect(listURL).toContain("source_id=issue-1");
  expect(listURL).toContain("source_revision=sha256%3Aabc");
  await expect(page.getByText("Propose knowledge", { exact: true })).toHaveCount(0);
  await expect(page.getByText("Review queue", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "View details" }).click();
  await expect(page.getByText(/acceptance_conclusion:issue-1@sha256:abc/)).toBeVisible();
});
