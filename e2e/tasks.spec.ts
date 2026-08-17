import { expect, test, type Locator, type Page } from "@playwright/test";

const EMAIL = "canonical-fixture@multica.local";
const SLUG = "canonical-fixture";

async function loginFixture(page: Page) {
  await page.goto("/login", { waitUntil: "domcontentloaded" });
  await page.locator("#login-email").fill(EMAIL);
  await page.getByRole("button", { name: "Continue" }).click();
  await page.locator('input[autocomplete="one-time-code"]').fill("888888");
  await page.waitForURL(new RegExp(`/${SLUG}/issues`));
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

test("installed Task surface manages lifecycle, filtering, reorder, archive and restore", async ({ page }) => {
  const marker = Date.now().toString(36);
  const firstTitle = `S02A browser first ${marker}`;
  const editedTitle = `S02A browser edited ${marker}`;
  const secondTitle = `S02A browser second ${marker}`;

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

  await taskRow(page, secondTitle).getByRole("button", { name: "Move up" }).click();
  await expect(page.getByRole("listitem").first()).toContainText(secondTitle);

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
});
