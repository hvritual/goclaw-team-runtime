import { expect, test, type Page } from "@playwright/test";

const EMAIL = `skill-lifecycle-${Date.now()}@multica.local`;
const SLUG = `skill-lifecycle-${Date.now().toString(36)}`;
const SCREENSHOT = `${process.env.TEMP ?? process.env.TMP ?? "."}\\goclaw-s05a-skill-lifecycle.png`;

test.setTimeout(120_000);

async function installOwnerSession(page: Page) {
  const login = await page.request.post("/auth/verify-code", {
    data: { email: EMAIL, code: "888888" },
  });
  expect(login.status(), await login.text()).toBe(200);
  const { token } = (await login.json()) as { token: string };
  const headers = { Authorization: `Bearer ${token}` };
  const created = await page.request.post("/api/workspaces", {
    headers,
    data: { name: "Skill Lifecycle", slug: SLUG },
  });
  expect(created.status(), await created.text()).toBe(201);
  const workspace = (await created.json()) as { id: string };
  const completed = await page.request.post("/api/me/onboarding/complete", {
    headers,
    data: { completion_path: "full", workspace_id: workspace.id },
  });
  expect(completed.status(), await completed.text()).toBe(200);
  await page.addInitScript((value) => {
    localStorage.setItem("multica_token", value);
    localStorage.setItem("multica:chat:isOpen", "false");
  }, token);
}

test("installed Skill lifecycle creates immutable versions and restores an archive", async ({ page }) => {
  const skillErrors: string[] = [];
  page.on("console", (message) => {
    if (
      message.type() === "error" &&
      (message.text().includes("/api/skills") ||
        message.text().includes("api-schema"))
    ) {
      skillErrors.push(message.text());
    }
  });
  page.on("response", (response) => {
    if (response.url().includes("/api/skills") && response.status() >= 400) {
      skillErrors.push(`${response.status()} ${response.url()}`);
    }
  });

  await installOwnerSession(page);
  await page.goto(`/${SLUG}/skills`, { waitUntil: "domcontentloaded" });
  await expect(page.getByRole("button", { name: "New skill" })).toBeVisible({
    timeout: 30_000,
  });

  await page.getByRole("button", { name: "New skill" }).click();
  await page.getByRole("button", { name: "Create manually" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("Name").fill("release-helper");
  await dialog.getByLabel("Description").fill("First immutable version");
  await dialog.getByRole("button", { name: "Create skill" }).click();

  await expect(page).toHaveURL(new RegExp(`/${SLUG}/skills/[^/]+$`));
  const detailURL = page.url();
  await expect(page.getByText("Version 1 · draft", { exact: true })).toBeVisible();
  await page.getByLabel("Description").fill("Second immutable version");
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByText("Version 2 · draft", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Publish" }).click();
  await expect(page.getByText("Version 2 · published", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Deprecate" })).toBeVisible();

  await page.getByLabel("Archive skill").click();
  await page.getByRole("button", { name: "Archive skill" }).click();
  await expect(page).toHaveURL(new RegExp(`/${SLUG}/skills$`));
  await expect(page.getByText("release-helper", { exact: true })).toHaveCount(0);

  await page.goto(detailURL, { waitUntil: "domcontentloaded" });
  await expect(page.getByRole("button", { name: "Restore" })).toBeVisible();
  await page.getByRole("button", { name: "Restore" }).click();
  await expect(page.getByText("Version 2 · published", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Deprecate" })).toBeVisible();
  await expect(page.locator("nextjs-portal")).toHaveCount(0);
  expect(skillErrors).toEqual([]);
  await page.screenshot({ path: SCREENSHOT, fullPage: false });
});
