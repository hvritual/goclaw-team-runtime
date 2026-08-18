import { expect, test, type Page } from "@playwright/test";

const EMAIL = `skill-lifecycle-${Date.now()}@multica.local`;
const SLUG = `skill-lifecycle-${Date.now().toString(36)}`;
const SCREENSHOT = `${process.env.TEMP ?? process.env.TMP ?? "."}\\goclaw-s05a-skill-lifecycle.png`;
const IMPORT_SCREENSHOT = `${process.env.TEMP ?? process.env.TMP ?? "."}\\goclaw-s05b-skill-import.png`;

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

function crc32(value: Buffer) {
  let crc = 0xffffffff;
  for (const byte of value) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit++) crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0);
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function skillArchive(files: Record<string, string>) {
  const local: Buffer[] = [];
  const central: Buffer[] = [];
  let offset = 0;
  for (const [path, content] of Object.entries(files)) {
    const name = Buffer.from(path);
    const body = Buffer.from(content);
    const checksum = crc32(body);
    const header = Buffer.alloc(30);
    header.writeUInt32LE(0x04034b50, 0);
    header.writeUInt16LE(20, 4);
    header.writeUInt32LE(checksum, 14);
    header.writeUInt32LE(body.length, 18);
    header.writeUInt32LE(body.length, 22);
    header.writeUInt16LE(name.length, 26);
    local.push(header, name, body);
    const directory = Buffer.alloc(46);
    directory.writeUInt32LE(0x02014b50, 0);
    directory.writeUInt16LE(20, 4);
    directory.writeUInt16LE(20, 6);
    directory.writeUInt32LE(checksum, 16);
    directory.writeUInt32LE(body.length, 20);
    directory.writeUInt32LE(body.length, 24);
    directory.writeUInt16LE(name.length, 28);
    directory.writeUInt32LE(offset, 42);
    central.push(directory, name);
    offset += header.length + name.length + body.length;
  }
  const centralBody = Buffer.concat(central);
  const end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50, 0);
  end.writeUInt16LE(Object.keys(files).length, 8);
  end.writeUInt16LE(Object.keys(files).length, 10);
  end.writeUInt32LE(centralBody.length, 12);
  end.writeUInt32LE(offset, 16);
  return Buffer.concat([...local, centralBody, end]);
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
  await expect(page.getByRole("button", { name: "Import from URL" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Import archive" })).toBeVisible();
  await page.getByRole("button", { name: "Create manually" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("Name").fill("release-helper");
  await dialog.getByLabel("Description").fill("First immutable version");
  await dialog.getByRole("button", { name: "Create skill" }).click();

  await expect(page).toHaveURL(new RegExp(`/${SLUG}/skills/[^/]+$`));
  const detailURL = page.url();
  await expect(page.getByText("Version 1 · draft", { exact: true })).toBeVisible();
  await expect(page.getByText("Audit activity", { exact: true })).toBeVisible();
  await expect(page.getByText("skill.created", { exact: true })).toBeVisible();
  await page.getByLabel("Description").fill("Second immutable version");
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByText("Version 3 · draft", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Publish" }).click();
  await expect(page.getByText("Version 3 · published", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Deprecate" })).toBeVisible();

  await page.getByLabel("Archive skill").click();
  await page.getByRole("button", { name: "Archive skill" }).click();
  await expect(page).toHaveURL(new RegExp(`/${SLUG}/skills$`));
  await expect(page.getByText("release-helper", { exact: true })).toHaveCount(0);

  await page.goto(detailURL, { waitUntil: "domcontentloaded" });
  await expect(page.getByRole("button", { name: "Restore" })).toBeVisible();
  await page.getByRole("button", { name: "Restore" }).click();
  await expect(page.getByText("Version 3 · published", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Deprecate" })).toBeVisible();
  await expect(page.locator("nextjs-portal")).toHaveCount(0);
  expect(skillErrors).toEqual([]);
  await page.screenshot({ path: SCREENSHOT, fullPage: false });

  await page.goto(`/${SLUG}/skills`, { waitUntil: "domcontentloaded" });
  await page.getByRole("button", { name: "New skill" }).click();
  await page.getByRole("button", { name: "Import archive" }).click();
  await page.getByLabel("Skill archive").setInputFiles({
    name: "governed-helper.skill",
    mimeType: "application/zip",
    buffer: skillArchive({
      "SKILL.md": "---\nname: Governed Helper\ndescription: Imported with preview\n---\n# Governed Helper",
      "scripts/run.py": "print('governed')\n",
    }),
  });
  await page.getByRole("button", { name: "Preview" }).click();
  await expect(page.getByTestId("skill-import-preview")).toContainText("Governed Helper");
  await expect(page.getByTestId("skill-import-preview")).toContainText("scripts/run.py");
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/${SLUG}/skills/[^/]+$`));
  await expect(page.getByText("Version 1 · draft", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "run.py", exact: true }).click();
  await expect(page.locator("textarea").last()).toHaveValue("print('governed')\n");
  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download file" }).click();
  await expect((await downloadPromise).suggestedFilename()).toBe("run.py");
  expect(skillErrors).toEqual([]);
  await page.screenshot({ path: IMPORT_SCREENSHOT, fullPage: false });
});
