import { expect, test, type Page } from "@playwright/test";

const RUN_ID = Date.now().toString(36);
const EMAIL = `knowledge-review-${RUN_ID}@multica.local`;
const SLUG = `knowledge-review-${RUN_ID}`;

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
    data: { name: "Knowledge Review", slug: SLUG },
  });
  expect(created.status(), await created.text()).toBe(201);
  const workspace = (await created.json()) as { id: string };
  const onboarded = await page.request.post("/api/me/onboarding/complete", {
    headers,
    data: { completion_path: "full", workspace_id: workspace.id },
  });
  expect(onboarded.status(), await onboarded.text()).toBe(200);
  await page.addInitScript((value) => {
    localStorage.setItem("multica_token", value);
    localStorage.setItem("multica:chat:isOpen", "false");
  }, token);
}

test("owner emergency review publishes governed knowledge through the real runtime", async ({
  page,
}) => {
  const consoleErrors: string[] = [];
  const httpErrors: string[] = [];
  page.on("console", (message) => {
    if (
      message.type() === "error" &&
      !message.text().startsWith("Failed to load resource:") &&
      !message.text().includes("ws://127.0.0.1:3106/ws")
    ) {
      consoleErrors.push(message.text());
    }
  });
  page.on("pageerror", (error) => consoleErrors.push(error.message));
  page.on("response", (response) => {
    if (response.status() >= 400) {
      httpErrors.push(
        `${response.status()} ${new URL(response.url()).pathname}`
      );
    }
  });

  await installOwnerSession(page);
  await page.goto(`/${SLUG}/knowledge`, { waitUntil: "domcontentloaded" });
  await expect(page.getByRole("heading", { name: "Knowledge" })).toBeVisible({
    timeout: 30_000,
  });
  await expect(page.getByText("Review queue", { exact: true })).toBeVisible();

  const title = `Governed browser lesson ${RUN_ID}`;
  await page.getByRole("button", { name: "Propose knowledge" }).click();
  await page.getByPlaceholder("A concise title").fill(title);
  await page
    .getByPlaceholder("What should the team retain?")
    .fill("Retain exact revisions and an auditable decision trail.");
  await page
    .getByPlaceholder("Why should this become governed knowledge?")
    .fill("The browser acceptance establishes the governed lifecycle.");
  await page.getByPlaceholder("Source ID").fill(`browser-${RUN_ID}`);
  await page
    .getByPlaceholder("Source revision")
    .fill(`sha256:${"a".repeat(64)}`);
  await page
    .getByPlaceholder("Citation")
    .fill("Release 2 browser acceptance conclusion");
  const proposalResponse = page.waitForResponse(
    (response) =>
      response.request().method() === "POST" &&
      new URL(response.url()).pathname === "/api/knowledge/proposals"
  );
  await page.getByRole("button", { name: "Submit proposal" }).click();
  expect((await proposalResponse).status()).toBe(201);

  const candidate = page
    .getByTestId("knowledge-review-queue")
    .getByRole("article")
    .filter({ hasText: title });
  await expect(candidate).toBeVisible();
  await candidate
    .getByPlaceholder("Add a review rationale")
    .fill("Emergency owner acceptance for isolated browser verification.");
  await candidate
    .getByText("Emergency self-review with documented reason")
    .click();

  const approveResponse = page.waitForResponse(
    (response) =>
      response.request().method() === "POST" &&
      new URL(response.url()).pathname.endsWith("/review")
  );
  await candidate.getByRole("button", { name: "approve" }).click();
  expect((await approveResponse).status()).toBe(200);
  await expect(
    candidate.getByRole("button", { name: "publish" })
  ).toBeVisible();

  const publishResponse = page.waitForResponse(
    (response) =>
      response.request().method() === "POST" &&
      new URL(response.url()).pathname.endsWith("/review")
  );
  await candidate.getByRole("button", { name: "publish" }).click();
  expect((await publishResponse).status()).toBe(200);

  await expect(candidate).toHaveCount(0);
  await expect(page.getByText(title, { exact: true })).toBeVisible();
  expect(consoleErrors).toEqual([]);
  expect(
    httpErrors.filter((entry) => entry !== "404 /api/invitations")
  ).toEqual([]);
});
