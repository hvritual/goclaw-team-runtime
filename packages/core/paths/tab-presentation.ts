import type { IssueStatus } from "../types";
import {
  WORKSPACE_PAGES,
  type NavLabelKey,
  type RouteIconName,
} from "./route-icons";
import type { TabActorType, TabSubject } from "./tab-subject";

export type TabVisual =
  | { kind: "icon"; icon: RouteIconName }
  | { kind: "issue-status"; status: IssueStatus | null }
  | { kind: "project-icon"; icon: string | null }
  | { kind: "actor"; actorType: TabActorType; id: string };

export type TabLabelKey =
  | "issue"
  | "project"
  | "task"
  | "member"
  | "skill"
  | "attachment"
  | "unknown";

export type TabTitleSpec =
  | { kind: "text"; text: string }
  | { kind: "nav"; navKey: NavLabelKey }
  | { kind: "tab"; tabKey: TabLabelKey };

export interface TabPresentation {
  visual: TabVisual;
  title: TabTitleSpec;
}

export interface TabEntityData {
  issue?: { identifier: string; title: string; status: IssueStatus };
  project?: { icon: string | null; title: string };
  task?: { title: string };
  actorName?: string;
  skill?: { name: string };
}

export const DEFAULT_TAB_VISUAL: TabVisual = {
  kind: "icon",
  icon: "FileQuestion",
};

function textOr(
  text: string | undefined | null,
  tabKey: TabLabelKey,
): TabTitleSpec {
  const trimmed = text?.trim();
  return trimmed ? { kind: "text", text: trimmed } : { kind: "tab", tabKey };
}

const EXTENSION_ICON: Record<string, RouteIconName> = {};
const registerExtensions = (icon: RouteIconName, extensions: string[]) => {
  for (const extension of extensions) EXTENSION_ICON[extension] = icon;
};
registerExtensions("FileImage", [
  "png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "ico", "avif", "heic",
]);
registerExtensions("FileVideo", ["mp4", "mov", "webm", "mkv", "avi", "m4v"]);
registerExtensions("FileAudio", ["mp3", "wav", "ogg", "flac", "m4a", "aac"]);
registerExtensions("FileArchive", ["zip", "tar", "gz", "tgz", "rar", "7z", "bz2"]);
registerExtensions("FileCode", [
  "js", "jsx", "ts", "tsx", "json", "yaml", "yml", "py", "go", "rs",
  "java", "c", "h", "cpp", "cc", "rb", "php", "swift", "kt", "sh", "css", "scss",
]);
registerExtensions("FileText", [
  "txt", "md", "markdown", "pdf", "doc", "docx", "csv", "log",
  "html", "htm", "xml", "rtf", "odt",
]);

export function iconForAttachment(filename: string | null): RouteIconName {
  if (!filename) return "File";
  const dot = filename.lastIndexOf(".");
  if (dot < 0 || dot === filename.length - 1) return "File";
  return EXTENSION_ICON[filename.slice(dot + 1).toLowerCase()] ?? "File";
}

export function resolveTabPresentation(
  subject: TabSubject,
  data: TabEntityData = {},
): TabPresentation {
  switch (subject.kind) {
    case "page": {
      const page = WORKSPACE_PAGES[subject.page];
      return {
        visual: { kind: "icon", icon: page.icon },
        title: { kind: "nav", navKey: page.navKey },
      };
    }
    case "issue":
      return {
        visual: { kind: "issue-status", status: data.issue?.status ?? null },
        title: data.issue
          ? { kind: "text", text: `${data.issue.identifier}: ${data.issue.title}` }
          : { kind: "tab", tabKey: "issue" },
      };
    case "project":
      return {
        visual: { kind: "project-icon", icon: data.project?.icon ?? null },
        title: textOr(data.project?.title, "project"),
      };
    case "task":
      return {
        visual: { kind: "icon", icon: "ListChecks" },
        title: textOr(data.task?.title, "task"),
      };
    case "actor":
      return {
        visual: { kind: "actor", actorType: subject.actorType, id: subject.id },
        title: textOr(data.actorName, "member"),
      };
    case "skill":
      return {
        visual: { kind: "icon", icon: "BookOpenText" },
        title: textOr(data.skill?.name, "skill"),
      };
    case "attachment":
      return {
        visual: { kind: "icon", icon: iconForAttachment(subject.filename) },
        title: subject.filename
          ? { kind: "text", text: subject.filename }
          : { kind: "tab", tabKey: "attachment" },
      };
    case "unknown":
      return {
        visual: DEFAULT_TAB_VISUAL,
        title: { kind: "tab", tabKey: "unknown" },
      };
  }
}
