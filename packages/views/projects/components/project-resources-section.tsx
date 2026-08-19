"use client";

import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowDown,
  ArrowUp,
  ChevronRight,
  FolderGit,
  Link2,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  Trash2,
  X,
} from "lucide-react";
import { toast } from "sonner";
import { ApiError } from "@multica/core/api";
import {
  projectResourcesOptions,
  useCreateProjectResource,
  useDeleteProjectResource,
  useUpdateProjectResource,
} from "@multica/core/projects";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCurrentWorkspace } from "@multica/core/paths";
import type {
  CreateProjectResourceRequest,
  GithubRepoResourceRef,
  KnownProjectResourceType,
  ProjectResource,
} from "@multica/core/types";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@multica/ui/components/ui/popover";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@multica/ui/components/ui/tooltip";
import { useT } from "../../i18n";
import { githubShortLabel } from "../../common/github-url";

type ResourceMutation = ReturnType<typeof useUpdateProjectResource>;
const EMPTY_PROJECT_RESOURCES: ProjectResource[] = [];

function resourceURL(resource: ProjectResource): string | null {
  const value = resource.resource_ref.url;
  return typeof value === "string" && value.startsWith("https://")
    ? value
    : null;
}

function resourceDisplay(resource: ProjectResource): string {
  if (resource.label) return resource.label;
  if (
    resource.resource_type !== "github_repo" &&
    resource.resource_type !== "url"
  ) {
    return resource.resource_type;
  }
  const url = resourceURL(resource);
  if (!url) return resource.resource_type;
  if (resource.resource_type === "github_repo") {
    const reference = resource.resource_ref as GithubRepoResourceRef;
    return reference.ref
      ? `${githubShortLabel(url)} @ ${reference.ref}`
      : githubShortLabel(url);
  }
  return url;
}

export function ProjectResourcesSection({
  projectId,
  canManage,
}: {
  projectId: string;
  canManage: boolean;
}) {
  const { t } = useT("projects");
  const workspaceId = useWorkspaceId();
  const workspace = useCurrentWorkspace();
  const [open, setOpen] = useState(true);
  const [addOpen, setAddOpen] = useState(false);
  const query = useQuery(
    projectResourcesOptions(workspaceId, projectId, true),
  );
  const createResource = useCreateProjectResource(workspaceId, projectId);
  const updateResource = useUpdateProjectResource(workspaceId, projectId);
  const archiveResource = useDeleteProjectResource(workspaceId, projectId);
  const resources = query.data?.resources ?? EMPTY_PROJECT_RESOURCES;
  const revision = query.data?.revision ?? 0;
  const activeResources = useMemo(
    () => resources.filter((resource) => resource.status === "active"),
    [resources],
  );
  const archivedResources = useMemo(
    () => resources.filter((resource) => resource.status === "archived"),
    [resources],
  );
  const attachedURLs = useMemo(
    () =>
      new Set(
        resources
          .map(resourceURL)
          .filter((value): value is string => value !== null),
      ),
    [resources],
  );

  const update = async (
    resource: ProjectResource,
    data: Parameters<ResourceMutation["mutateAsync"]>[0]["data"],
  ) => {
    try {
      await updateResource.mutateAsync({ resourceId: resource.id, data });
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t(($) => $.resources.toast_update_failed),
      );
    }
  };

  const archive = async (resource: ProjectResource) => {
    try {
      await archiveResource.mutateAsync({
        resourceId: resource.id,
        expectedRevision: revision,
      });
      toast.success(t(($) => $.resources.toast_removed));
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t(($) => $.resources.toast_remove_failed),
      );
    }
  };

  const create = async (request: CreateProjectResourceRequest) => {
    try {
      await createResource.mutateAsync(request);
      toast.success(t(($) => $.resources.toast_attached));
      setAddOpen(false);
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t(($) => $.resources.toast_attach_failed),
      );
      throw error;
    }
  };

  return (
    <section aria-labelledby="project-resources-heading">
      <button
        type="button"
        className={`mb-2 flex w-full items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors hover:bg-accent/70 ${
          open ? "" : "text-muted-foreground hover:text-foreground"
        }`}
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
      >
        <span id="project-resources-heading">
          {t(($) => $.resources.section_header)}
        </span>
        <ChevronRight
          aria-hidden="true"
          className={`size-3 shrink-0 stroke-[2.5] text-muted-foreground transition-transform ${
            open ? "rotate-90" : ""
          }`}
        />
      </button>

      {open ? (
        <div className="space-y-3 pl-2">
          {query.isLoading ? (
            <p className="text-xs text-muted-foreground" aria-live="polite">
              {t(($) => $.resources.loading)}
            </p>
          ) : query.error ? (
            <p className="text-xs text-destructive" role="alert">
              {query.error instanceof ApiError && query.error.status === 403
                ? t(($) => $.resources.denied)
                : t(($) => $.resources.load_failed)}
            </p>
          ) : (
            <>
              {resources.length === 0 ? (
                <p className="text-xs text-muted-foreground">
                  {t(($) => $.resources.empty)}
                </p>
              ) : null}

              {activeResources.length > 0 ? (
                <div className="space-y-1.5">
                  {activeResources.map((resource, index) => (
                    <ResourceRow
                      key={resource.id}
                      resource={resource}
                      canManage={canManage}
                      revision={revision}
                      activeIndex={index}
                      activeResources={activeResources}
                      update={update}
                      archive={archive}
                    />
                  ))}
                </div>
              ) : null}

              {archivedResources.length > 0 ? (
                <div className="space-y-1.5 border-t pt-2">
                  <p className="px-1 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
                    {t(($) => $.resources.archived_group)}
                  </p>
                  {archivedResources.map((resource) => (
                    <ResourceRow
                      key={resource.id}
                      resource={resource}
                      canManage={canManage}
                      revision={revision}
                      activeIndex={-1}
                      activeResources={activeResources}
                      update={update}
                      archive={archive}
                    />
                  ))}
                </div>
              ) : null}
            </>
          )}

          {canManage && !query.error ? (
            <Popover open={addOpen} onOpenChange={setAddOpen}>
              <PopoverTrigger
                render={
                  <button
                    type="button"
                    className="inline-flex h-7 items-center gap-1 rounded-md px-2 text-xs text-muted-foreground hover:bg-accent hover:text-foreground"
                  >
                    <Plus aria-hidden="true" className="size-3" />
                    {t(($) => $.resources.add_button)}
                  </button>
                }
              />
              <PopoverContent align="start" className="w-80 space-y-3 p-3">
                <ResourceCreateForm
                  repos={workspace?.repos?.map((repo) => repo.url) ?? []}
                  attachedURLs={attachedURLs}
                  pending={createResource.isPending}
                  onCreate={create}
                />
              </PopoverContent>
            </Popover>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

function ResourceRow({
  resource,
  canManage,
  revision,
  activeIndex,
  activeResources,
  update,
  archive,
}: {
  resource: ProjectResource;
  canManage: boolean;
  revision: number;
  activeIndex: number;
  activeResources: ProjectResource[];
  update: (
    resource: ProjectResource,
    data: Parameters<ResourceMutation["mutateAsync"]>[0]["data"],
  ) => Promise<void>;
  archive: (resource: ProjectResource) => Promise<void>;
}) {
  const { t } = useT("projects");
  const [editing, setEditing] = useState(false);
  const [label, setLabel] = useState(resource.label ?? "");
  const display = resourceDisplay(resource);
  const url = resourceURL(resource);
  const isGitHub = resource.resource_type === "github_repo";
  const isKnownURL = isGitHub || resource.resource_type === "url";
  const Icon = isGitHub ? FolderGit : Link2;
  const connectionLabel = t(
    ($) => $.resources.connection[resource.connection.state],
  );

  const reorder = async (direction: "up" | "down") => {
    const before =
      direction === "up"
        ? activeResources[activeIndex - 1]?.id
        : activeResources[activeIndex + 2]?.id;
    await update(resource, {
      action: "reorder",
      expected_revision: revision,
      ...(before ? { before_resource_id: before } : {}),
    });
  };

  return (
    <article className="group rounded-md border border-transparent px-1.5 py-1.5 hover:border-border hover:bg-accent/30">
      <div className="flex min-w-0 items-center gap-2 text-xs">
        <Icon
          aria-hidden="true"
          className="size-3.5 shrink-0 text-muted-foreground"
        />
        <div className="min-w-0 flex-1">
          {url && isKnownURL ? (
            <Tooltip>
              <TooltipTrigger
                render={
                  <a
                    href={url}
                    target="_blank"
                    rel="noopener noreferrer"
                    aria-label={display}
                    className="block truncate font-medium hover:underline"
                  >
                    {display}
                  </a>
                }
              />
              <TooltipContent side="top" className="max-w-sm break-all">
                {url}
              </TooltipContent>
            </Tooltip>
          ) : (
            <span className="block truncate font-medium">{display}</span>
          )}
          <div className="mt-0.5 flex flex-wrap items-center gap-1.5 text-[10px] text-muted-foreground">
            {!isKnownURL ? (
              <span className="rounded bg-muted px-1 py-0.5">
                {resource.resource_type}
              </span>
            ) : null}
            <span
              className="rounded bg-muted px-1 py-0.5"
              title={resource.connection.diagnostic_code}
            >
              {connectionLabel}
            </span>
            {resource.status === "archived" ? (
              <span className="rounded bg-muted px-1 py-0.5">
                {t(($) => $.resources.archived)}
              </span>
            ) : null}
          </div>
        </div>

        {canManage ? (
          <div className="flex shrink-0 items-center gap-0.5">
            {resource.status === "active" ? (
              <>
                <IconButton
                  label={t(($) => $.resources.move_up_aria, {
                    name: display,
                  })}
                  disabled={activeIndex <= 0}
                  onClick={() => reorder("up")}
                >
                  <ArrowUp className="size-3" />
                </IconButton>
                <IconButton
                  label={t(($) => $.resources.move_down_aria, {
                    name: display,
                  })}
                  disabled={
                    activeIndex < 0 || activeIndex >= activeResources.length - 1
                  }
                  onClick={() => reorder("down")}
                >
                  <ArrowDown className="size-3" />
                </IconButton>
                <IconButton
                  label={t(($) => $.resources.refresh_aria, {
                    name: display,
                  })}
                  onClick={() =>
                    update(resource, {
                      action: "refresh",
                      expected_revision: revision,
                    })
                  }
                >
                  <RefreshCw className="size-3" />
                </IconButton>
                <IconButton
                  label={t(($) => $.resources.edit_label_aria, {
                    name: display,
                  })}
                  onClick={() => setEditing(true)}
                >
                  <Pencil className="size-3" />
                </IconButton>
                <IconButton
                  label={t(($) => $.resources.archive_aria, {
                    name: display,
                  })}
                  onClick={() => archive(resource)}
                >
                  <Trash2 className="size-3" />
                </IconButton>
              </>
            ) : (
              <IconButton
                label={t(($) => $.resources.restore_aria, { name: display })}
                onClick={() =>
                  update(resource, {
                    action: "restore",
                    expected_revision: revision,
                  })
                }
              >
                <RotateCcw className="size-3" />
              </IconButton>
            )}
          </div>
        ) : null}
      </div>

      {editing ? (
        <form
          className="mt-2 flex items-center gap-1.5 pl-5"
          onSubmit={async (event) => {
            event.preventDefault();
            await update(resource, {
              action: "update",
              expected_revision: revision,
              label: label.trim(),
            });
            setEditing(false);
          }}
        >
          <input
            aria-label={t(($) => $.resources.label_input)}
            value={label}
            maxLength={120}
            onChange={(event) => setLabel(event.target.value)}
            className="h-7 min-w-0 flex-1 rounded-md border bg-transparent px-2 text-xs outline-none focus-visible:ring-1 focus-visible:ring-ring"
          />
          <IconButton
            label={t(($) => $.resources.save_label)}
            type="submit"
          >
            <Save className="size-3" />
          </IconButton>
          <IconButton
            label={t(($) => $.resources.cancel_edit)}
            onClick={() => setEditing(false)}
          >
            <X className="size-3" />
          </IconButton>
        </form>
      ) : null}
    </article>
  );
}

function IconButton({
  label,
  children,
  type = "button",
  ...props
}: {
  label: string;
  children: React.ReactNode;
  type?: "button" | "submit";
} & Omit<
  React.ButtonHTMLAttributes<HTMLButtonElement>,
  "aria-label" | "type"
>) {
  return (
    <button
      {...props}
      type={type}
      aria-label={label}
      className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground disabled:cursor-not-allowed disabled:opacity-30"
    >
      {children}
    </button>
  );
}

function ResourceCreateForm({
  repos,
  attachedURLs,
  pending,
  onCreate,
}: {
  repos: string[];
  attachedURLs: Set<string>;
  pending: boolean;
  onCreate: (request: CreateProjectResourceRequest) => Promise<void>;
}) {
  const { t } = useT("projects");
  const [type, setType] =
    useState<KnownProjectResourceType>("github_repo");
  const [url, setURL] = useState("");
  const [ref, setRef] = useState("");
  const [label, setLabel] = useState("");

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    const normalizedURL = url.trim();
    if (!normalizedURL) return;
    await onCreate({
      resource_type: type,
      resource_ref:
        type === "github_repo"
          ? {
              url: normalizedURL,
              ...(ref.trim() ? { ref: ref.trim() } : {}),
            }
          : { url: normalizedURL },
      ...(label.trim() ? { label: label.trim() } : {}),
    });
    setURL("");
    setRef("");
    setLabel("");
  };

  return (
    <div className="space-y-3">
      <p className="text-xs font-medium text-muted-foreground">
        {t(($) => $.resources.popover_title)}
      </p>
      {type === "github_repo" && repos.length > 0 ? (
        <div className="max-h-28 space-y-1 overflow-y-auto">
          {repos.map((repo) => {
            const attached = attachedURLs.has(repo);
            return (
              <button
                key={repo}
                type="button"
                disabled={attached || pending}
                aria-label={t(($) => $.resources.attach_repo_aria, {
                  url: repo,
                })}
                onClick={() => setURL(repo)}
                className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs hover:bg-accent disabled:opacity-50"
              >
                <FolderGit className="size-3.5 shrink-0" />
                <span className="min-w-0 flex-1 truncate">
                  {githubShortLabel(repo)}
                </span>
                {attached ? (
                  <span>{t(($) => $.resources.attached_badge)}</span>
                ) : null}
              </button>
            );
          })}
        </div>
      ) : null}
      <form className="space-y-2 border-t pt-2" onSubmit={submit}>
        <label className="block space-y-1 text-xs">
          <span className="text-muted-foreground">
            {t(($) => $.resources.type_input)}
          </span>
          <select
            aria-label={t(($) => $.resources.type_input)}
            value={type}
            onChange={(event) =>
              setType(event.target.value as KnownProjectResourceType)
            }
            className="h-8 w-full rounded-md border bg-background px-2"
          >
            <option value="github_repo">
              {t(($) => $.resources.type_github)}
            </option>
            <option value="url">{t(($) => $.resources.type_url)}</option>
          </select>
        </label>
        <label className="block space-y-1 text-xs">
          <span className="text-muted-foreground">
            {t(($) => $.resources.url_input)}
          </span>
          <input
            aria-label={t(($) => $.resources.url_input)}
            value={url}
            onChange={(event) => setURL(event.target.value)}
            placeholder={
              type === "github_repo"
                ? "https://github.com/owner/repo"
                : "https://example.com/docs"
            }
            className="h-8 w-full rounded-md border bg-transparent px-2 outline-none focus-visible:ring-1 focus-visible:ring-ring"
          />
        </label>
        {type === "github_repo" ? (
          <label className="block space-y-1 text-xs">
            <span className="text-muted-foreground">
              {t(($) => $.resources.ref_input)}
            </span>
            <input
              aria-label={t(($) => $.resources.ref_input)}
              value={ref}
              maxLength={255}
              onChange={(event) => setRef(event.target.value)}
              className="h-8 w-full rounded-md border bg-transparent px-2 outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
          </label>
        ) : null}
        <label className="block space-y-1 text-xs">
          <span className="text-muted-foreground">
            {t(($) => $.resources.label_input)}
          </span>
          <input
            aria-label={t(($) => $.resources.label_input)}
            value={label}
            maxLength={120}
            onChange={(event) => setLabel(event.target.value)}
            className="h-8 w-full rounded-md border bg-transparent px-2 outline-none focus-visible:ring-1 focus-visible:ring-ring"
          />
        </label>
        <button
          type="submit"
          disabled={!url.trim() || pending}
          className="inline-flex h-8 items-center gap-1 rounded-md bg-primary px-3 text-xs text-primary-foreground disabled:opacity-50"
        >
          <Plus className="size-3" />
          {type === "url"
            ? t(($) => $.resources.add_url)
            : t(($) => $.resources.add_github)}
        </button>
      </form>
    </div>
  );
}
