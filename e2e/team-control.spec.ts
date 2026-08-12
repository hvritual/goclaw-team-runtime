import { expect, test, type Page, type Route } from "@playwright/test";

const workspace = {
  id: "ws-1",
  name: "Acme",
  slug: "acme",
  description: null,
  context: null,
  settings: {},
  repos: [],
  issue_prefix: "ACME",
  avatar_url: null,
  created_at: "2026-08-12T00:00:00Z",
  updated_at: "2026-08-12T00:00:00Z",
};

const user = {
  id: "user-1",
  name: "Team Control Tester",
  email: "tester@example.test",
  avatar_url: null,
  onboarded_at: "2026-08-12T00:00:00Z",
  onboarding_questionnaire: {},
  starter_content_state: "imported",
  language: "en",
  profile_description: "",
  timezone: "UTC",
  created_at: "2026-08-12T00:00:00Z",
  updated_at: "2026-08-12T00:00:00Z",
};

function projection(head: number) {
  return {
    schema_version: 1,
    workspace_id: workspace.id,
    project_id: "project-1",
    head,
    head_hash: "a".repeat(64),
    nodes: {
      "requirement-1": {
        id: "requirement-1",
        kind: "requirement",
        revision: 1,
        state: "clarifying",
        creator_id: "member-1",
        assignee_ids: [],
        executor_ids: [],
        data: { request: "Preserve the audit trail" },
      },
    },
    edges: {},
    evidence: {},
    checks: {},
    acceptances: {},
  };
}

async function fulfillJson(route: Route, body: unknown, status = 200) {
  await route.fulfill({
    status,
    contentType: status >= 400 ? "application/problem+json" : "application/json",
    body: JSON.stringify(body),
  });
}

async function mockAppShell(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem("multica_token", "e2e-token");
    localStorage.setItem("multica:chat:isOpen", "false");
  });
  await page.route("**/api/**", async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === "/api/me") return fulfillJson(route, user);
    if (url.pathname === "/api/workspaces") return fulfillJson(route, [workspace]);
    if (url.pathname === "/api/config") return fulfillJson(route, {});
    if (url.pathname.endsWith("/members")) return fulfillJson(route, []);
    if (url.pathname.includes("/agents")) return fulfillJson(route, []);
    if (url.pathname.includes("/pins")) return fulfillJson(route, []);
    if (url.pathname.includes("/inbox")) return fulfillJson(route, []);
    if (url.pathname.includes("/chat/sessions")) return fulfillJson(route, []);
    if (url.pathname === "/api/squads") return fulfillJson(route, []);
    return fulfillJson(route, []);
  });
}

async function openTeamControl(page: Page) {
  const pageErrors: string[] = [];
  const consoleErrors: string[] = [];
  page.on("pageerror", (error) => pageErrors.push(error.message));
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  await page.goto("/acme/projects/project-1/control", { waitUntil: "domcontentloaded" });
  await expect.poll(() => pageErrors).toEqual([]);
  await expect.poll(() => consoleErrors).toEqual([]);
  await expect(page.getByRole("heading", { name: "Team Control" })).toBeVisible();
}

test.describe("Team Control", () => {
  test.describe.configure({ mode: "parallel" });

  test("renders the governed projection and refreshes a CAS conflict", async ({ page }) => {
    await mockAppShell(page);
    let head = 3;
    let projectionReads = 0;
    await page.route("**/control-plane/v1/**", async (route) => {
      const request = route.request();
      const url = new URL(request.url());
      if (url.pathname.endsWith("/events")) {
        return route.fulfill({ status: 200, contentType: "text/event-stream", body: ": heartbeat\n\n" });
      }
      if (url.pathname.endsWith("/members")) {
        return fulfillJson(route, {
          schema_version: 1,
          members: [{
            workspace_id: workspace.id,
            id: "member-1",
            kind: "human",
            role: "owner",
            state: "active",
            version: 1,
            created_at: "2026-08-12T00:00:00Z",
            updated_at: "2026-08-12T00:00:00Z",
          }],
        });
      }
      if (url.pathname.endsWith("/projects/project-1/projection") && request.method() === "GET") {
        projectionReads += 1;
        return fulfillJson(route, projection(head));
      }
      if (url.pathname.endsWith("/projects/project-1/commands")) {
        head = 4;
        return fulfillJson(route, {
          type: "about:blank",
          title: "Conflict",
          status: 409,
          code: "conflict",
          detail: "The project Head changed.",
        }, 409);
      }
      return fulfillJson(route, {
        schema_version: 1,
        workspace: {
          id: workspace.id,
          name: workspace.name,
          state: "active",
          version: 1,
          created_at: "2026-08-12T00:00:00Z",
          updated_at: "2026-08-12T00:00:00Z",
        },
      });
    });

    await openTeamControl(page);
    await expect(page.getByText("Preserve the audit trail")).toBeVisible();
    await expect(page.getByText("Head 3")).toBeVisible();

    await page.getByRole("tab", { name: "Overview" }).focus();
    await page.keyboard.press("ArrowRight");
    const requirementsTab = page.getByRole("tab", { name: "Requirements" });
    await expect(requirementsTab).toBeFocused();
    await requirementsTab.click();
    await page.getByRole("button", { name: "Start requirement" }).click();
    const dialog = page.getByRole("dialog", { name: "Start requirement" });
    await expect(dialog).toBeVisible();
    await dialog.getByLabel("ID").fill("requirement-2");
    await dialog.getByLabel("Summary").fill("Add a safe retry policy");
    await dialog.getByRole("button", { name: "Submit command" }).click();

    await expect(dialog.getByText("Authoritative state changed")).toBeVisible();
    await expect(dialog.getByLabel("Summary")).toHaveValue("Add a safe retry policy");
    await expect(page.getByText("Head 4")).toBeVisible();
    expect(projectionReads).toBeGreaterThan(1);
  });

  test("uses SSE to refresh only the visible project projection", async ({ page }) => {
    await mockAppShell(page);
    let projectionReads = 0;
    await page.route("**/control-plane/v1/**", async (route) => {
      const url = new URL(route.request().url());
      if (url.pathname.endsWith("/events")) {
        const event = {
          schema_version: 1,
          workspace_id: workspace.id,
          project_id: "project-1",
          sequence: 4,
          event_id: "event-4",
          command_id: "command-4",
          type: "requirement.started",
          actor_id: "member-1",
          actor_kind: "human",
          payload: {},
          previous_hash: "a".repeat(64),
          hash: "b".repeat(64),
          occurred_at: "2026-08-12T00:00:00Z",
        };
        await new Promise((resolve) => setTimeout(resolve, 300));
        return route.fulfill({
          status: 200,
          contentType: "text/event-stream",
          body: `id: 4\nevent: session\ndata: ${JSON.stringify(event)}\n\n`,
        });
      }
      if (url.pathname.endsWith("/projects/project-1/projection")) {
        projectionReads += 1;
        return fulfillJson(route, projection(projectionReads === 1 ? 3 : 4));
      }
      if (url.pathname.endsWith("/members")) {
        return fulfillJson(route, { schema_version: 1, members: [] });
      }
      return fulfillJson(route, {
        schema_version: 1,
        workspace: {
          id: workspace.id,
          name: workspace.name,
          state: "active",
          version: 1,
          created_at: "2026-08-12T00:00:00Z",
          updated_at: "2026-08-12T00:00:00Z",
        },
      });
    });

    await openTeamControl(page);
    await expect(page.getByText("Head 4")).toBeVisible();
    expect(projectionReads).toBeGreaterThan(1);
  });

  test("renders a non-enumerating denied state without mobile overflow", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await mockAppShell(page);
    await page.route("**/control-plane/v1/**", async (route) => {
      const url = new URL(route.request().url());
      if (url.pathname.endsWith("/projects/project-1/projection")) {
        return fulfillJson(route, {
          type: "about:blank",
          title: "Forbidden",
          status: 403,
          code: "denied",
          detail: "Denied",
        }, 403);
      }
      if (url.pathname.endsWith("/events")) {
        return route.fulfill({ status: 403, contentType: "text/plain", body: "" });
      }
      if (url.pathname.endsWith("/members")) {
        return fulfillJson(route, { schema_version: 1, members: [] });
      }
      return fulfillJson(route, {
        schema_version: 1,
        workspace: {
          id: workspace.id,
          name: workspace.name,
          state: "active",
          version: 1,
          created_at: "2026-08-12T00:00:00Z",
          updated_at: "2026-08-12T00:00:00Z",
        },
      });
    });

    const pageErrors: string[] = [];
    const consoleErrors: string[] = [];
    page.on("pageerror", (error) => pageErrors.push(error.message));
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    await page.goto("/acme/projects/project-1/control", { waitUntil: "domcontentloaded" });
    await expect.poll(() => pageErrors).toEqual([]);
    await expect.poll(() => consoleErrors).toEqual([]);
    await expect(page.getByText("Team Control access denied")).toBeVisible();
    await expect(page.getByText("Preserve the audit trail")).toHaveCount(0);
    const hasOverflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth);
    expect(hasOverflow).toBe(false);
  });
});
