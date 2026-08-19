import { DatabaseSync } from "node:sqlite";
import { expect, test, type BrowserContext, type Page } from "@playwright/test";

const RUN_ID = Date.now().toString(36);
const OWNER_EMAIL = `project-resource-owner-${RUN_ID}@example.com`;
const MEMBER_EMAIL = `project-resource-member-${RUN_ID}@example.com`;
const SLUG = `project-resources-${RUN_ID}`;
const BASE_URL =
  process.env.PLAYWRIGHT_BASE_URL ??
  process.env.FRONTEND_ORIGIN ??
  "http://127.0.0.1:3000";
const BROWSER_404_ERROR =
  "Failed to load resource: the server responded with a status of 404 (Not Found)";

test.setTimeout(120_000);

type Login = { token: string; user: { id: string } };
type Project = { id: string; resource_count: number };
type Resource = {
  id: string;
  label?: string;
  resource_type: string;
  revision: number;
};
type ResourceList = { resources: Resource[]; total: number; revision: number };

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

async function useEnglish(context: BrowserContext) {
  await context.addCookies([
    { name: "multica-locale", value: "en", url: BASE_URL },
  ]);
}

function resourceRow(page: Page, name: string) {
  return page
    .getByRole("article")
    .filter({ has: page.getByRole("link", { name, exact: true }) });
}

test("installed Project Resources enforce the real owner, member, and lead journey", async ({
  browser,
  page: ownerPage,
}) => {
  const sqlitePath = process.env.E2E_SQLITE_PATH?.trim();
  if (!sqlitePath) throw new Error("E2E_SQLITE_PATH is required");

  const owner = await login(ownerPage, OWNER_EMAIL);
  const ownerHeaders = { Authorization: `Bearer ${owner.token}` };
  const workspaceResponse = await ownerPage.request.post("/api/workspaces", {
    headers: ownerHeaders,
    data: { name: `Project Resources ${RUN_ID}`, slug: SLUG },
  });
  expect(workspaceResponse.status(), await workspaceResponse.text()).toBe(201);
  const workspace = (await workspaceResponse.json()) as { id: string };
  const ownerOnboarded = await ownerPage.request.post(
    "/api/me/onboarding/complete",
    {
      headers: ownerHeaders,
      data: { completion_path: "full", workspace_id: workspace.id },
    },
  );
  expect(ownerOnboarded.status(), await ownerOnboarded.text()).toBe(200);

  const member = await login(ownerPage, MEMBER_EMAIL);
  const memberRowID = crypto.randomUUID();
  const database = new DatabaseSync(sqlitePath);
  try {
    // The backend and this fixture share the same rollback-journal database.
    // Wait for a just-finished HTTP write to release its transient lock instead
    // of making the acceptance result depend on scheduler timing.
    database.exec("PRAGMA busy_timeout = 5000");
    database
      .prepare(
        "INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES(?,?,?,?,?)",
      )
      .run(
        memberRowID,
        workspace.id,
        member.user.id,
        "member",
        new Date().toISOString(),
      );
  } finally {
    database.close();
  }
  const memberHeaders = { Authorization: `Bearer ${member.token}` };
  const memberOnboarded = await ownerPage.request.post(
    "/api/me/onboarding/complete",
    {
      headers: memberHeaders,
      data: { completion_path: "full", workspace_id: workspace.id },
    },
  );
  expect(memberOnboarded.status(), await memberOnboarded.text()).toBe(200);

  await useEnglish(ownerPage.context());
  await installSession(ownerPage, owner.token);
  const consoleErrors: string[] = [];
  const httpErrors: string[] = [];
  ownerPage.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  ownerPage.on("pageerror", (error) => consoleErrors.push(error.message));
  ownerPage.on("response", (response) => {
    if (response.status() >= 400) {
      httpErrors.push(`${response.status()} ${new URL(response.url()).pathname}`);
    }
  });

  await ownerPage.goto(`/${SLUG}/projects`, {
    waitUntil: "domcontentloaded",
  });
  try {
    await expect(
      ownerPage.getByRole("heading", { name: "Projects", exact: true }),
    ).toBeVisible({ timeout: 30_000 });
  } catch (error) {
    console.log({
      url: ownerPage.url(),
      title: await ownerPage.title(),
      body: await ownerPage.locator("body").innerText(),
      consoleErrors,
      httpErrors,
    });
    throw error;
  }
  await ownerPage.getByRole("button", { name: "New project" }).click();
  await ownerPage
    .getByRole("textbox", { name: "Project title", exact: true })
    .fill("S07A Runtime");
  await ownerPage.getByRole("button", { name: "Repos" }).click();
  await ownerPage
    .getByPlaceholder(
      "https://github.com/owner/repo or git@github.com:owner/repo.git",
    )
    .fill("git@github.com:Acme/Runtime.git");
  await ownerPage.getByRole("button", { name: "Add", exact: true }).click();
  await ownerPage
    .getByRole("button", { name: "Create Project", exact: true })
    .click();
  await ownerPage.waitForURL(new RegExp(`/${SLUG}/projects/[^/]+$`), {
    timeout: 30_000,
  });
  const projectID = new URL(ownerPage.url()).pathname.split("/").at(-1);
  if (!projectID) throw new Error("Project detail URL did not contain an ID");

  await expect(
    ownerPage.getByRole("button", { name: "Resources", exact: true }),
  ).toBeVisible({ timeout: 30_000 });
  await expect(
    ownerPage.getByRole("link", { name: "acme/runtime", exact: true }),
  ).toBeVisible();
  await expect(resourceRow(ownerPage, "acme/runtime")).toContainText(
    "Not checked",
  );

  await ownerPage
    .getByRole("button", { name: "Add resource", exact: true })
    .click();
  await ownerPage
    .getByRole("combobox", { name: "Resource type", exact: true })
    .selectOption("url");
  await ownerPage
    .getByRole("textbox", { name: "Resource URL", exact: true })
    .fill("https://example.com/runbook");
  await ownerPage
    .getByRole("textbox", {
      name: "Display label (optional)",
      exact: true,
    })
    .fill("Runbook");
  await ownerPage
    .getByRole("button", { name: "Add URL", exact: true })
    .click();
  await expect(
    ownerPage.getByRole("link", { name: "Runbook", exact: true }),
  ).toBeVisible();

  await ownerPage
    .getByRole("button", { name: "Refresh Runbook", exact: true })
    .click();
  await expect(resourceRow(ownerPage, "Runbook")).toContainText(
    "Unavailable",
  );

  const ownerWorkspaceHeaders = {
    ...ownerHeaders,
    "X-Workspace-Slug": SLUG,
  };
  const listResponse = await ownerPage.request.get(
    `/api/projects/${projectID}/resources?include_archived=true`,
    { headers: ownerWorkspaceHeaders },
  );
  expect(listResponse.status(), await listResponse.text()).toBe(200);
  const listed = (await listResponse.json()) as ResourceList;
  const runbook = listed.resources.find((resource) => resource.label === "Runbook");
  expect(runbook).toBeDefined();
  const staleReorder = await ownerPage.request.put(
    `/api/projects/${projectID}/resources/${runbook!.id}`,
    {
      headers: ownerWorkspaceHeaders,
      data: { action: "reorder", expected_revision: listed.revision - 1 },
    },
  );
  expect(staleReorder.status(), await staleReorder.text()).toBe(409);
  await expect(staleReorder.json()).resolves.toMatchObject({
    code: "revision_conflict",
    current_revision: listed.revision,
  });

  await ownerPage
    .getByRole("button", { name: "Archive Runbook", exact: true })
    .click();
  await expect(ownerPage.getByText("Archived resources", { exact: true })).toBeVisible();
  await expect(resourceRow(ownerPage, "Runbook")).toContainText("Archived");
  await ownerPage.reload({ waitUntil: "domcontentloaded" });
  await expect(resourceRow(ownerPage, "Runbook")).toContainText("Archived", {
    timeout: 30_000,
  });
  await ownerPage
    .getByRole("button", { name: "Restore Runbook", exact: true })
    .click();
  await expect(resourceRow(ownerPage, "Runbook")).not.toContainText("Archived");

  const memberContext = await browser.newContext({ baseURL: BASE_URL });
  await useEnglish(memberContext);
  const memberPage = await memberContext.newPage();
  await installSession(memberPage, member.token);
  memberPage.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  memberPage.on("pageerror", (error) => consoleErrors.push(error.message));
  memberPage.on("response", (response) => {
    if (response.status() >= 400) {
      httpErrors.push(`${response.status()} ${new URL(response.url()).pathname}`);
    }
  });
  try {
    await memberPage.goto(`/${SLUG}/projects/${projectID}`, {
      waitUntil: "domcontentloaded",
    });
    await expect(
      memberPage.getByRole("link", { name: "Runbook", exact: true }),
    ).toBeVisible({ timeout: 30_000 });
    await expect(
      memberPage.getByRole("button", { name: "Add resource", exact: true }),
    ).toHaveCount(0);
    await expect(
      memberPage.getByRole("button", { name: "Archive Runbook", exact: true }),
    ).toHaveCount(0);

    const assignLead = await ownerPage.request.put(
      `/api/projects/${projectID}`,
      {
        headers: ownerWorkspaceHeaders,
        data: { lead_type: "member", lead_id: member.user.id },
      },
    );
    expect(assignLead.status(), await assignLead.text()).toBe(200);
    await memberPage.reload({ waitUntil: "domcontentloaded" });
    await expect(
      memberPage.getByRole("button", { name: "Add resource", exact: true }),
    ).toBeVisible({ timeout: 30_000 });
    await memberPage
      .getByRole("button", { name: "Add resource", exact: true })
      .click();
    await memberPage
      .getByRole("combobox", { name: "Resource type", exact: true })
      .selectOption("url");
    await memberPage
      .getByRole("textbox", { name: "Resource URL", exact: true })
      .fill("https://example.com/lead-managed");
    await memberPage
      .getByRole("textbox", {
        name: "Display label (optional)",
        exact: true,
      })
      .fill("Lead managed");
    await memberPage
      .getByRole("button", { name: "Add URL", exact: true })
      .click();
    await expect(
      memberPage.getByRole("link", { name: "Lead managed", exact: true }),
    ).toBeVisible();
  } finally {
    await memberContext.close();
  }

  const projectResponse = await ownerPage.request.get(
    `/api/projects/${projectID}`,
    { headers: ownerWorkspaceHeaders },
  );
  expect(projectResponse.status(), await projectResponse.text()).toBe(200);
  const project = (await projectResponse.json()) as Project;
  expect(project.resource_count).toBe(3);

  const deleteProject = await ownerPage.request.delete(
    `/api/projects/${projectID}`,
    { headers: ownerWorkspaceHeaders },
  );
  expect(deleteProject.status(), await deleteProject.text()).toBe(204);
  const deletedResources = await ownerPage.request.get(
    `/api/projects/${projectID}/resources?include_archived=true`,
    { headers: ownerWorkspaceHeaders },
  );
  expect(deletedResources.status(), await deletedResources.text()).toBe(404);
  const deletedDatabase = new DatabaseSync(sqlitePath, { readOnly: true });
  try {
    const remainingResources = deletedDatabase
      .prepare(
        "SELECT COUNT(*) AS count FROM workspace_project_resources WHERE project_id = ?",
      )
      .get(projectID) as { count: number };
    const remainingSet = deletedDatabase
      .prepare(
        "SELECT COUNT(*) AS count FROM workspace_project_resource_sets WHERE project_id = ?",
      )
      .get(projectID) as { count: number };
    expect(remainingResources.count).toBe(0);
    expect(remainingSet.count).toBe(0);
  } finally {
    deletedDatabase.close();
  }

  const knownInvitation404s = httpErrors.filter(
    (entry) => entry === "404 /api/invitations",
  );
  const browser404s = consoleErrors.filter((entry) => entry === BROWSER_404_ERROR);
  expect(httpErrors.filter((entry) => entry !== "404 /api/invitations")).toEqual(
    [],
  );
  expect(consoleErrors.filter((entry) => entry !== BROWSER_404_ERROR)).toEqual(
    [],
  );
  expect(browser404s).toHaveLength(knownInvitation404s.length);
});
