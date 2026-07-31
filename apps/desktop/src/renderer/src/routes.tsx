import { useEffect } from "react";
import { createMemoryRouter, Outlet, useMatches } from "react-router-dom";
import type { RouteObject } from "react-router-dom";
import { IssueDetailPage } from "./pages/issue-detail-page";
import { ProjectDetailPage } from "./pages/project-detail-page";
import { SkillDetailPage } from "./pages/skill-detail-page";
import { MemberDetailPage } from "./pages/member-detail-page";
import { AttachmentPreviewRoute } from "./pages/attachment-preview-page";
import { IssuesPage } from "@multica/views/issues/components";
import { ProjectsPage } from "@multica/views/projects/components";
import { TasksPage } from "@multica/views/tasks";
import { KnowledgePage } from "@multica/views/knowledge";
import { MyIssuesPage } from "@multica/views/my-issues";
import { SkillsPage } from "@multica/views/skills";
import { SettingsPage } from "@multica/views/settings";
import { useT } from "@multica/views/i18n";
import { Download } from "lucide-react";
import { UpdatesSettingsTab } from "./components/updates-settings-tab";
import { WorkspaceRouteLayout } from "./components/workspace-route-layout";
import { DesktopRouteErrorPage } from "./components/route-error-page";

/**
 * Wraps `SettingsPage` so the desktop-only extra tabs can pull their labels
 * from i18n. The route element has to be a component (not a literal JSX
 * value) for `useT` to run.
 */
function DesktopSettingsRoute() {
  const { t } = useT("settings");
  return (
    <SettingsPage
      extraAccountTabs={[
        {
          value: "updates",
          label: t(($) => $.desktop.tabs.updates),
          icon: Download,
          content: <UpdatesSettingsTab />,
        },
      ]}
    />
  );
}

/**
 * Sets document.title from the deepest matched route's handle.title.
 * The tab system observes document.title via MutationObserver.
 * Pages with dynamic titles (e.g. issue detail) override by setting
 * document.title directly via useDocumentTitle().
 */
function TitleSync() {
  const matches = useMatches();
  const title = [...matches]
    .reverse()
    .find((m) => (m.handle as { title?: string })?.title)
    ?.handle as { title?: string } | undefined;

  useEffect(() => {
    if (title?.title) document.title = title.title;
  }, [title?.title]);

  return null;
}

/** Wrapper that renders route children + TitleSync */
function PageShell() {
  return (
    <>
      <TitleSync />
      <Outlet />
    </>
  );
}

/**
 * Route definitions shared by all tabs.
 *
 * Every tab path is workspace-scoped: `/{slug}/{route}/...`. Pre-workspace
 * flows (create workspace, accept invite) are NOT routes — they render as a
 * window-level overlay via `WindowOverlay`, dispatched by the navigation
 * adapter's transition-path interception. The `activeWorkspaceSlug` in the
 * tab store decides which workspace's tabs are visible in the TabBar;
 * workspace-less state (zero-workspace user) shows the overlay instead.
 *
 * The root index route stays as a harmless safety net. With per-workspace
 * tabs, nothing should construct a tab at `/` — but if one ever slips
 * through (malformed persisted state that dodges the migration, direct
 * router.navigate from unforeseen code), the index falls back to null
 * rather than 404; App.tsx's bootstrap repoints activeWorkspaceSlug on the
 * next render pass.
 */
export const appRoutes: RouteObject[] = [
  {
    element: <PageShell />,
    errorElement: <DesktopRouteErrorPage />,
    children: [
      { index: true, element: null },
      {
        path: ":workspaceSlug",
        element: <WorkspaceRouteLayout />,
        children: [
          // A bare `/{slug}` URL is normalized to `/{slug}/issues` by
          // sanitizeTabPath before it ever becomes a session, so the index
          // route is unreachable in practice; null keeps it a harmless
          // safety net instead of an in-router <Navigate> (MUL-4741
          // invariant 1: the router never self-navigates).
          { index: true, element: null },
          {
            path: "issues",
            element: <IssuesPage />,
            handle: { title: "Issues" },
          },
          {
            path: "issues/:id",
            element: <IssueDetailPage />,
            handle: { title: "Issue" },
          },
          {
            path: "projects",
            element: <ProjectsPage />,
            handle: { title: "Projects" },
          },
          {
            path: "projects/:id",
            element: <ProjectDetailPage />,
            handle: { title: "Project" },
          },
          {
            path: "tasks",
            element: <TasksPage />,
            handle: { title: "Tasks" },
          },
          {
            path: "knowledge",
            element: <KnowledgePage />,
            handle: { title: "Knowledge" },
          },
          {
            path: "my-issues",
            element: <MyIssuesPage />,
            handle: { title: "My Issues" },
          },
          { path: "skills", element: <SkillsPage />, handle: { title: "Skills" } },
          {
            path: "skills/:id",
            element: <SkillDetailPage />,
            handle: { title: "Skill" },
          },
          {
            path: "members/:id",
            element: <MemberDetailPage />,
            handle: { title: "Member" },
          },
          {
            path: "attachments/:id/preview",
            element: <AttachmentPreviewRoute />,
            handle: { title: "Attachment" },
          },
          {
            path: "settings",
            element: <DesktopSettingsRoute />,
            handle: { title: "Settings" },
          },
        ],
      },
    ],
  },
];

/**
 * Create THE app router (MUL-4741 single-router session architecture).
 * There is exactly one instance, owned by the tab Coordinator; it projects
 * the active tab session's URL and is never navigated by anything else.
 */
export function createAppRouter() {
  return createMemoryRouter(appRoutes, {
    initialEntries: ["/"],
  });
}
