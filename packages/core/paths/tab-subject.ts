import { pageForSegment, type WorkspacePageKey } from "./route-icons";

export type TabActorType = "member";

export type TabSubject =
  | { kind: "page"; page: WorkspacePageKey }
  | { kind: "issue"; id: string }
  | { kind: "project"; id: string }
  | { kind: "task"; id: string }
  | { kind: "actor"; actorType: TabActorType; id: string }
  | { kind: "skill"; id: string }
  | { kind: "attachment"; id: string; filename: string | null }
  | { kind: "unknown" };

function splitUrl(url: string): { segments: string[]; query: URLSearchParams } {
  const hashIndex = url.indexOf("#");
  const withoutHash = hashIndex === -1 ? url : url.slice(0, hashIndex);
  const queryIndex = withoutHash.indexOf("?");
  const pathname =
    queryIndex === -1 ? withoutHash : withoutHash.slice(0, queryIndex);
  const search =
    queryIndex === -1 ? "" : withoutHash.slice(queryIndex + 1);
  return {
    segments: pathname.split("/").filter(Boolean),
    query: new URLSearchParams(search),
  };
}

export function parseTabSubject(url: string): TabSubject {
  const { segments, query } = splitUrl(url);
  const segment = segments[1] ?? "";
  const id = segments[2] ?? "";

  switch (segment) {
    case "issues":
      return id ? { kind: "issue", id } : { kind: "page", page: "issues" };
    case "my-issues":
      return { kind: "page", page: "myIssues" };
    case "projects":
      return id
        ? { kind: "project", id }
        : { kind: "page", page: "projects" };
    case "tasks":
      return id ? { kind: "task", id } : { kind: "page", page: "tasks" };
    case "members":
      return id
        ? { kind: "actor", actorType: "member", id }
        : { kind: "unknown" };
    case "skills":
      return id ? { kind: "skill", id } : { kind: "page", page: "skills" };
    case "settings":
      return { kind: "page", page: "settings" };
    case "attachments":
      return id
        ? { kind: "attachment", id, filename: query.get("name") || null }
        : { kind: "unknown" };
    default: {
      const page = pageForSegment(segment);
      return page ? { kind: "page", page } : { kind: "unknown" };
    }
  }
}

export function tabSubjectKey(subject: TabSubject): string {
  switch (subject.kind) {
    case "page":
      return `page:${subject.page}`;
    case "issue":
      return `issue:${subject.id}`;
    case "project":
      return `project:${subject.id}`;
    case "task":
      return `task:${subject.id}`;
    case "actor":
      return `actor:${subject.actorType}:${subject.id}`;
    case "skill":
      return `skill:${subject.id}`;
    case "attachment":
      return `attachment:${subject.id}:${subject.filename ?? ""}`;
    case "unknown":
      return "unknown";
  }
}
