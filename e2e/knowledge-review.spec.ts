import { DatabaseSync } from "node:sqlite";
import { expect, test, type Page } from "@playwright/test";

const RUN_ID = Date.now().toString(36);
const OWNER_EMAIL = `knowledge-owner-${RUN_ID}@multica.local`;
const MEMBER_EMAIL = `knowledge-member-${RUN_ID}@multica.local`;
const SLUG = `knowledge-review-${RUN_ID}`;

test.setTimeout(120_000);

type Login = { token: string; user: { id: string } };

async function login(page: Page, email: string): Promise<Login> {
  const response = await page.request.post("/auth/verify-code", {
    data: { email, code: "888888" },
  });
  expect(response.status(), await response.text()).toBe(200);
  return (await response.json()) as Login;
}

async function installSession(page: Page, token: string) {
  await page.addInitScript((value) => {
    localStorage.setItem("multica_token", value);
    localStorage.setItem("multica:chat:isOpen", "false");
  }, token);
}

function captureUnexpectedErrors(
  page: Page,
  consoleErrors: string[],
  httpErrors: string[]
) {
  page.on("console", (message) => {
    if (
      message.type() === "error" &&
      !message.text().startsWith("Failed to load resource:")
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
}

test("member proposal reaches an independent owner and publishes through the real runtime", async ({
  browser,
  page: memberPage,
}) => {
  const sqlitePath = process.env.E2E_SQLITE_PATH?.trim();
  if (!sqlitePath) throw new Error("E2E_SQLITE_PATH is required");

  const owner = await login(memberPage, OWNER_EMAIL);
  const ownerHeaders = { Authorization: `Bearer ${owner.token}` };
  const created = await memberPage.request.post("/api/workspaces", {
    headers: ownerHeaders,
    data: { name: "Knowledge Review", slug: SLUG },
  });
  expect(created.status(), await created.text()).toBe(201);
  const workspace = (await created.json()) as { id: string };
  const ownerOnboarded = await memberPage.request.post(
    "/api/me/onboarding/complete",
    {
      headers: ownerHeaders,
      data: { completion_path: "full", workspace_id: workspace.id },
    }
  );
  expect(ownerOnboarded.status(), await ownerOnboarded.text()).toBe(200);

  const member = await login(memberPage, MEMBER_EMAIL);
  const database = new DatabaseSync(sqlitePath);
  try {
    database
      .prepare(
        "INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES(?,?,?,?,?)"
      )
      .run(
        crypto.randomUUID(),
        workspace.id,
        member.user.id,
        "member",
        new Date().toISOString()
      );
  } finally {
    database.close();
  }
  const memberHeaders = { Authorization: `Bearer ${member.token}` };
  const memberOnboarded = await memberPage.request.post(
    "/api/me/onboarding/complete",
    {
      headers: memberHeaders,
      data: { completion_path: "full", workspace_id: workspace.id },
    }
  );
  expect(memberOnboarded.status(), await memberOnboarded.text()).toBe(200);

  await installSession(memberPage, member.token);
  const ownerContext = await browser.newContext({
    baseURL: process.env.PLAYWRIGHT_BASE_URL,
  });
  const ownerPage = await ownerContext.newPage();
  await installSession(ownerPage, owner.token);
  const consoleErrors: string[] = [];
  const httpErrors: string[] = [];
  captureUnexpectedErrors(memberPage, consoleErrors, httpErrors);
  captureUnexpectedErrors(ownerPage, consoleErrors, httpErrors);

  try {
    await Promise.all([
      memberPage.goto(`/${SLUG}/knowledge`, { waitUntil: "domcontentloaded" }),
      ownerPage.goto(`/${SLUG}/knowledge`, { waitUntil: "domcontentloaded" }),
    ]);
    await expect(
      memberPage.getByRole("heading", { name: "Knowledge" })
    ).toBeVisible({
      timeout: 30_000,
    });
    await expect(
      memberPage.getByText("Review queue", { exact: true })
    ).toHaveCount(0);
    await expect(
      ownerPage.getByText("Review queue", { exact: true })
    ).toBeVisible({
      timeout: 30_000,
    });

    const title = `Governed browser lesson ${RUN_ID}`;
    await memberPage.getByRole("button", { name: "Propose knowledge" }).click();
    await memberPage.getByPlaceholder("A concise title").fill(title);
    await memberPage
      .getByPlaceholder("What should the team retain?")
      .fill("Retain exact revisions and an auditable decision trail.");
    await memberPage
      .getByPlaceholder("Why should this become governed knowledge?")
      .fill("The browser acceptance establishes the governed lifecycle.");
    await memberPage.getByPlaceholder("Source ID").fill(`browser-${RUN_ID}`);
    await memberPage
      .getByPlaceholder("Source revision")
      .fill(`sha256:${"a".repeat(64)}`);
    await memberPage
      .getByPlaceholder("Citation")
      .fill("Release 2 browser acceptance conclusion");
    const proposalResponse = memberPage.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        new URL(response.url()).pathname === "/api/knowledge/proposals"
    );
    await memberPage.getByRole("button", { name: "Submit proposal" }).click();
    expect((await proposalResponse).status()).toBe(201);

    const candidate = ownerPage
      .getByTestId("knowledge-review-queue")
      .getByRole("article")
      .filter({ hasText: title });
    await expect(candidate).toBeVisible({ timeout: 30_000 });
    await expect(
      candidate.getByText("Your proposal", { exact: true })
    ).toHaveCount(0);
    await candidate
      .getByPlaceholder("Add a review rationale")
      .fill("Independent owner acceptance from the browser journey.");
    await expect(
      candidate.getByText("Emergency self-review with documented reason")
    ).toHaveCount(0);

    const approveResponse = ownerPage.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        new URL(response.url()).pathname.endsWith("/review")
    );
    await candidate.getByRole("button", { name: "approve" }).click();
    expect((await approveResponse).status()).toBe(200);
    await expect(
      candidate.getByRole("button", { name: "publish" })
    ).toBeVisible();

    const publishResponse = ownerPage.waitForResponse(
      (response) =>
        response.request().method() === "POST" &&
        new URL(response.url()).pathname.endsWith("/review")
    );
    await candidate.getByRole("button", { name: "publish" }).click();
    expect((await publishResponse).status()).toBe(200);

    await expect(candidate).toHaveCount(0);
    await expect(memberPage.getByText(title, { exact: true })).toBeVisible({
      timeout: 30_000,
    });
    expect(consoleErrors).toEqual([]);
    expect(
      httpErrors.filter((entry) => entry !== "404 /api/invitations")
    ).toEqual([]);
  } finally {
    await ownerContext.close();
  }
});
