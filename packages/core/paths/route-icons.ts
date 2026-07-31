export type RouteIconName =
  | "CircleUser"
  | "ListTodo"
  | "FolderKanban"
  | "ListChecks"
  | "BookOpenText"
  | "LibraryBig"
  | "Settings"
  | "File"
  | "FileText"
  | "FileImage"
  | "FileCode"
  | "FileArchive"
  | "FileAudio"
  | "FileVideo"
  | "FileQuestion";

export type NavLabelKey =
  | "my_issues"
  | "issues"
  | "projects"
  | "tasks"
  | "knowledge"
  | "skills"
  | "settings";

export type WorkspacePageKey =
  | "myIssues"
  | "issues"
  | "projects"
  | "tasks"
  | "knowledge"
  | "skills"
  | "settings";

export interface WorkspacePage {
  segment: string;
  icon: RouteIconName;
  navKey: NavLabelKey;
}

export const WORKSPACE_PAGES: Record<WorkspacePageKey, WorkspacePage> = {
  myIssues: { segment: "my-issues", icon: "CircleUser", navKey: "my_issues" },
  issues: { segment: "issues", icon: "ListTodo", navKey: "issues" },
  projects: { segment: "projects", icon: "FolderKanban", navKey: "projects" },
  tasks: { segment: "tasks", icon: "ListChecks", navKey: "tasks" },
  knowledge: { segment: "knowledge", icon: "LibraryBig", navKey: "knowledge" },
  skills: { segment: "skills", icon: "BookOpenText", navKey: "skills" },
  settings: { segment: "settings", icon: "Settings", navKey: "settings" },
};

const PAGE_BY_SEGMENT: Record<string, WorkspacePageKey> = Object.fromEntries(
  (Object.keys(WORKSPACE_PAGES) as WorkspacePageKey[]).map((key) => [
    WORKSPACE_PAGES[key].segment,
    key,
  ]),
);

export function pageForSegment(segment: string): WorkspacePageKey | null {
  return PAGE_BY_SEGMENT[segment] ?? null;
}

export const DEFAULT_ROUTE_ICON_NAME: RouteIconName = "ListTodo";

export function resolveRouteIconName(path: string): RouteIconName {
  const pathname = path.split(/[?#]/)[0] ?? "";
  const segment = pathname.split("/").filter(Boolean)[1] ?? "";
  const page = pageForSegment(segment);
  return page ? WORKSPACE_PAGES[page].icon : DEFAULT_ROUTE_ICON_NAME;
}
