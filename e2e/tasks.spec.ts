import { expect, test, type Locator, type Page } from "@playwright/test";

const EMAIL = "canonical-fixture@multica.local";
const SLUG = "canonical-fixture";

test.setTimeout(120_000);

async function loginFixture(page: Page) {
	const response = await page.request.post("/auth/verify-code", {
		data: { email: EMAIL, code: "888888" },
	});
	expect(response.status()).toBe(200);
}

function taskRow(page: Page, title: string): Locator {
  return page.getByRole("listitem").filter({ hasText: title });
}

async function createTask(page: Page, title: string) {
  await page.getByPlaceholder("What needs to be done?").fill(title);
  await page.getByRole("button", { name: "Add task" }).click();
  await expect(taskRow(page, title)).toBeVisible();
}

async function setStatus(row: Locator, currentLabel: string, status: string, nextLabel: string) {
  await row.getByRole("combobox", { name: currentLabel }).selectOption(status);
  await expect(row.getByRole("combobox", { name: nextLabel })).toBeVisible();
}

test("installed Task surface manages lifecycle and promotes a Task to an Issue", async ({ page }, testInfo) => {
  const consoleErrors: string[] = [];
  const httpErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().startsWith("Failed to load resource:")) consoleErrors.push(message.text());
  });
  page.on("pageerror", (error) => consoleErrors.push(error.message));
  page.on("response", (response) => {
    if (response.status() >= 400) httpErrors.push(`${response.status()} ${new URL(response.url()).pathname}`);
  });
  const marker = Date.now().toString(36);
  const firstTitle = `S02A browser first ${marker}`;
  const editedTitle = `S02A browser edited ${marker}`;
  const secondTitle = `S02A browser second ${marker}`;
  const promotionTitle = `S02B browser promotion ${marker}`;

  await loginFixture(page);
  await page.goto(`/${SLUG}/tasks`, { waitUntil: "domcontentloaded" });
  await expect(page.getByRole("heading", { name: "Tasks" })).toBeVisible();

  await createTask(page, firstTitle);
  await createTask(page, secondTitle);

  const firstRow = taskRow(page, firstTitle);
  await firstRow.getByText("Edit task", { exact: true }).click();
  await firstRow.getByLabel("Task title").fill(editedTitle);
  await firstRow.getByLabel("Due date").fill("2026-08-25");
  await firstRow.getByRole("button", { name: "Save task" }).click();
  await expect(taskRow(page, editedTitle)).toContainText("Due Aug 25, 2026");

  const reorderResponse = page.waitForResponse((response) => response.url().endsWith("/api/tasks/reorder"));
  await taskRow(page, editedTitle).getByRole("button", { name: "Move up" }).click();
  const reorderResult = await reorderResponse;
  expect(reorderResult.status(), await reorderResult.text()).toBe(200);
  await expect.poll(async () => {
    const rows = await page.locator("main").getByRole("listitem").allTextContents();
    return rows.findIndex((text) => text.includes(editedTitle)) < rows.findIndex((text) => text.includes(secondTitle));
  }).toBe(true);

  await setStatus(taskRow(page, editedTitle), "To do", "in_progress", "In progress");
  await setStatus(taskRow(page, editedTitle), "In progress", "done", "Done");
  await setStatus(taskRow(page, secondTitle), "To do", "cancelled", "Cancelled");

  await page.getByLabel("Filter tasks by status").selectOption("done");
  await expect(taskRow(page, editedTitle)).toBeVisible();
  await expect(taskRow(page, secondTitle)).toHaveCount(0);
  await page.getByLabel("Filter tasks by status").selectOption("");

  await taskRow(page, editedTitle).getByRole("button", { name: "Archive" }).click();
  await taskRow(page, secondTitle).getByRole("button", { name: "Archive" }).click();
  await page.getByLabel("Filter tasks by status").selectOption("archived");
  await expect(taskRow(page, editedTitle)).toBeVisible();
  await taskRow(page, editedTitle).getByRole("button", { name: "Restore" }).click();
  await expect(taskRow(page, editedTitle)).toHaveCount(0);

  await page.getByLabel("Filter tasks by status").selectOption("");
  await createTask(page, promotionTitle);
  await setStatus(taskRow(page, promotionTitle), "To do", "in_progress", "In progress");
  const promotionRow = taskRow(page, promotionTitle);
  await promotionRow.getByRole("checkbox", { name: "Complete task after promotion" }).check();
  const promoteResponse = page.waitForResponse((response) => response.url().endsWith("/promote"));
  await promotionRow.getByRole("button", { name: "Promote to issue" }).click();
  const promoteResult = await promoteResponse;
  expect(promoteResult.status(), await promoteResult.text()).toBe(201);
  const issueLink = promotionRow.getByRole("link", { name: /^CAN-\d+$/ });
  await expect(issueLink).toBeVisible();
  await expect(promotionRow.getByRole("combobox", { name: "Done" })).toBeVisible();
  await issueLink.click();
  await expect(page).toHaveURL(new RegExp(`/${SLUG}/issues/[^/]+$`), { timeout: 30_000 });
  expect(consoleErrors).toEqual([]);
  expect(httpErrors.filter((entry) => entry !== "404 /api/invitations")).toEqual([]);
  await page.screenshot({ path: testInfo.outputPath("tasks-archived-filter.png"), fullPage: false });
});
