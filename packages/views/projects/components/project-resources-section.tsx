"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight, FolderGit, Plus, Search, Trash2 } from "lucide-react";
import { toast } from "sonner";
import {
  projectResourcesOptions,
  useCreateProjectResource,
  useDeleteProjectResource,
} from "@multica/core/projects";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCurrentWorkspace } from "@multica/core/paths";
import type {
  GithubRepoResourceRef,
  ProjectResource,
} from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
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

function isGithubResource(
  resource: ProjectResource,
): resource is ProjectResource & { resource_ref: GithubRepoResourceRef } {
  return resource.resource_type === "github_repo";
}

export function ProjectResourcesSection({ projectId }: { projectId: string }) {
  const { t } = useT("projects");
  const workspaceId = useWorkspaceId();
  const workspace = useCurrentWorkspace();
  const [open, setOpen] = useState(true);
  const [addOpen, setAddOpen] = useState(false);
  const [repoSearch, setRepoSearch] = useState("");
  const { data: resources = [] } = useQuery(
    projectResourcesOptions(workspaceId, projectId),
  );
  const createResource = useCreateProjectResource(workspaceId, projectId);
  const deleteResource = useDeleteProjectResource(workspaceId, projectId);

  const attachedUrls = new Set(
    resources.filter(isGithubResource).map((resource) => resource.resource_ref.url),
  );
  const normalizedSearch = repoSearch.trim().toLowerCase();
  const filteredRepos =
    workspace?.repos?.filter((repo) =>
      repo.url.toLowerCase().includes(normalizedSearch),
    ) ?? [];

  const attach = async (url: string) => {
    try {
      await createResource.mutateAsync({
        resource_type: "github_repo",
        resource_ref: { url },
      });
      toast.success(t(($) => $.resources.toast_attached));
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t(($) => $.resources.toast_attach_failed),
      );
    }
  };

  const remove = async (resource: ProjectResource) => {
    try {
      await deleteResource.mutateAsync(resource.id);
      toast.success(t(($) => $.resources.toast_removed));
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t(($) => $.resources.toast_remove_failed),
      );
    }
  };

  return (
    <div>
      <button
        type="button"
        className={`mb-2 flex w-full items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors hover:bg-accent/70 ${
          open ? "" : "text-muted-foreground hover:text-foreground"
        }`}
        onClick={() => setOpen((value) => !value)}
      >
        {t(($) => $.resources.section_header)}
        <ChevronRight
          className={`!size-3 shrink-0 stroke-[2.5] text-muted-foreground transition-transform ${
            open ? "rotate-90" : ""
          }`}
        />
      </button>
      {open && (
        <div className="space-y-1.5 pl-2">
          {resources.length === 0 && (
            <p className="text-xs text-muted-foreground">
              {t(($) => $.resources.empty)}
            </p>
          )}
          {resources.map((resource) => (
            <ResourceRow
              key={resource.id}
              resource={resource}
              onRemove={() => remove(resource)}
            />
          ))}
          <Popover
            open={addOpen}
            onOpenChange={(value) => {
              setAddOpen(value);
              if (!value) setRepoSearch("");
            }}
          >
            <PopoverTrigger
              render={
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
                >
                  <Plus className="size-3" />
                  {t(($) => $.resources.add_button)}
                </Button>
              }
            />
            <PopoverContent align="start" className="w-72 space-y-2 p-2">
              <div className="text-xs font-medium text-muted-foreground">
                {t(($) => $.resources.popover_title)}
              </div>
              {workspace?.repos && workspace.repos.length > 0 && (
                <>
                  <div className="relative">
                    <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                    <input
                      type="text"
                      value={repoSearch}
                      onChange={(event) => setRepoSearch(event.target.value)}
                      aria-label={t(($) => $.resources.repos_search_placeholder)}
                      placeholder={t(($) => $.resources.repos_search_placeholder)}
                      className="h-8 w-full rounded-md border bg-transparent pl-7 pr-2 text-xs outline-none placeholder:text-muted-foreground focus-visible:ring-1 focus-visible:ring-ring"
                    />
                  </div>
                  <div className="max-h-48 space-y-1 overflow-y-auto">
                    {filteredRepos.length === 0 && normalizedSearch && (
                      <p className="py-2 text-center text-xs text-muted-foreground">
                        {t(($) => $.resources.repos_search_empty)}
                      </p>
                    )}
                    {filteredRepos.map((repo) => {
                      const attached = attachedUrls.has(repo.url);
                      const disabled = attached || createResource.isPending;
                      return (
                        <button
                          key={repo.url}
                          type="button"
                          aria-disabled={disabled}
                          onClick={async () => {
                            if (disabled) return;
                            await attach(repo.url);
                            setAddOpen(false);
                          }}
                          className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs transition-colors hover:bg-accent aria-disabled:cursor-not-allowed aria-disabled:opacity-50 aria-disabled:hover:bg-transparent"
                        >
                          <FolderGit className="size-3.5" />
                          <span className="min-w-0 flex-1 truncate">
                            {githubShortLabel(repo.url)}
                          </span>
                          {attached && (
                            <span className="text-[10px] text-muted-foreground">
                              {t(($) => $.resources.attached_badge)}
                            </span>
                          )}
                        </button>
                      );
                    })}
                  </div>
                </>
              )}
              <CustomRepoForm
                onSubmit={async (url) => {
                  await attach(url);
                  setAddOpen(false);
                }}
              />
            </PopoverContent>
          </Popover>
        </div>
      )}
    </div>
  );
}

function ResourceRow({
  resource,
  onRemove,
}: {
  resource: ProjectResource;
  onRemove: () => void;
}) {
  const { t } = useT("projects");
  if (!isGithubResource(resource)) return null;
  const reference = resource.resource_ref;
  const display =
    resource.label ||
    (reference.ref
      ? `${githubShortLabel(reference.url)} @ ${reference.ref}`
      : githubShortLabel(reference.url));
  const tooltip = reference.ref
    ? `${reference.url}\nref: ${reference.ref}`
    : reference.url;
  return (
    <div className="group flex items-center gap-2 text-xs">
      <FolderGit className="size-3.5 shrink-0 text-muted-foreground" />
      <Tooltip>
        <TooltipTrigger
          render={
            <a
              href={reference.url}
              target="_blank"
              rel="noopener noreferrer"
              className="min-w-0 flex-1 truncate hover:underline"
            >
              {display}
            </a>
          }
        />
        <TooltipContent side="top" className="whitespace-pre-line">
          {tooltip}
        </TooltipContent>
      </Tooltip>
      <button
        type="button"
        onClick={onRemove}
        className="rounded-sm p-0.5 opacity-0 transition-opacity hover:bg-accent group-hover:opacity-100"
        title={t(($) => $.resources.remove_tooltip)}
      >
        <Trash2 className="size-3 text-muted-foreground" />
      </button>
    </div>
  );
}

function CustomRepoForm({
  onSubmit,
}: {
  onSubmit: (url: string) => Promise<void> | void;
}) {
  const { t } = useT("projects");
  const [url, setUrl] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    const trimmed = url.trim();
    if (!trimmed) return;
    setSubmitting(true);
    try {
      await onSubmit(trimmed);
      setUrl("");
    } finally {
      setSubmitting(false);
    }
  };
  return (
    <form
      onSubmit={handleSubmit}
      className="flex items-center gap-1.5 border-t pt-1"
    >
      <input
        type="text"
        value={url}
        onChange={(event) => setUrl(event.target.value)}
        placeholder={t(($) => $.resources.url_placeholder)}
        className="flex-1 bg-transparent px-2 py-1 text-xs outline-none placeholder:text-muted-foreground"
      />
      <Button
        type="submit"
        size="sm"
        variant="ghost"
        className="h-6 px-2 text-xs"
        disabled={!url.trim() || submitting}
      >
        {t(($) => $.resources.url_submit)}
      </Button>
    </form>
  );
}
