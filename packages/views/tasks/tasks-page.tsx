"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertCircle, Archive, ArrowDown, ArrowUp, CheckSquare2, Loader2, Plus, RotateCcw } from "lucide-react";
import type { TaskActorType, TaskStatus } from "@multica/core/types/task";
import { useWorkspaceId } from "@multica/core/hooks";
import {
  taskListOptions,
  useCreateTask,
  useDeleteTask,
  useRestoreTask,
  useReorderTasks,
  useUpdateTask,
} from "@multica/core/tasks";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { useT } from "../i18n";

const STATUSES: TaskStatus[] = [
  "todo",
  "in_progress",
  "done",
  "cancelled",
  "archived",
];

type MutableTaskStatus = Exclude<TaskStatus, "archived">;

const NEXT_STATUSES: Record<TaskStatus, TaskStatus[]> = {
  todo: ["todo", "in_progress", "cancelled"],
  in_progress: ["in_progress", "done", "cancelled"],
  done: ["done"],
  cancelled: ["cancelled"],
  archived: ["archived"],
};

const dueDateFormatter = new Intl.DateTimeFormat("en", { month: "short", day: "numeric", year: "numeric", timeZone: "UTC" });

export function TasksPage() {
  const { t } = useT("tasks");
  const workspaceId = useWorkspaceId();
  const [statusFilter, setStatusFilter] = useState<TaskStatus | "">("");
  const taskQuery = useQuery(
    taskListOptions(workspaceId, statusFilter ? { status: statusFilter } : undefined),
  );
  const tasks = taskQuery.data ?? [];
  const createTask = useCreateTask();
  const updateTask = useUpdateTask();
  const deleteTask = useDeleteTask();
  const restoreTask = useRestoreTask();
  const reorderTasks = useReorderTasks();
  const [title, setTitle] = useState("");
  const mutationError = createTask.error ?? updateTask.error ?? deleteTask.error ?? restoreTask.error ?? reorderTasks.error;

  const submit = () => {
    const value = title.trim();
    if (!value) return;
    createTask.mutate(
      { title: value },
      { onSuccess: () => setTitle("") },
    );
  };

  const moveTask = (index: number, offset: -1 | 1) => {
    const target = index + offset;
    if (target < 0 || target >= tasks.length) return;
    const reordered = [...tasks];
    [reordered[index], reordered[target]] = [reordered[target]!, reordered[index]!];
    reorderTasks.mutate({
      items: reordered.map((task, itemIndex) => ({ id: task.id, position: (itemIndex + 1) * 10, expected_revision: task.revision })),
    });
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
        <CheckSquare2 className="size-4 text-muted-foreground" />
        <h1 className="text-sm font-semibold">{t(($) => $.title)}</h1>
      </header>

      <main className="mx-auto flex w-full max-w-4xl flex-1 flex-col gap-4 overflow-y-auto p-4 sm:p-6">
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") submit();
            }}
            placeholder={t(($) => $.new_placeholder)}
          />
          <Button
            onClick={submit}
            disabled={!title.trim() || createTask.isPending}
          >
            {createTask.isPending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Plus className="size-4" />
            )}
            {t(($) => $.create)}
          </Button>
          <select
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value as TaskStatus | "")}
            className="h-8 rounded-md border bg-background px-2 text-sm"
            aria-label={t(($) => $.filter_label)}
          >
            <option value="">{t(($) => $.all_statuses)}</option>
            {STATUSES.map((status) => <option key={status} value={status}>{t(($) => $.status[status])}</option>)}
          </select>
        </div>

        {mutationError ? (
          <div role="alert" className="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
            <AlertCircle className="size-4 shrink-0" />
            {String(mutationError).match(/409|revision.?conflict/i)
              ? t(($) => $.conflict)
              : t(($) => $.mutation_error)}
          </div>
        ) : null}

        {taskQuery.isLoading ? (
          <div className="flex flex-1 items-center justify-center">
            <Loader2 className="size-5 animate-spin text-muted-foreground" />
          </div>
        ) : taskQuery.isError ? (
          <div role="alert" className="flex flex-1 flex-col items-center justify-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-10 text-sm text-destructive">
            <AlertCircle className="size-8 opacity-60" />
            {String(taskQuery.error).match(/403|permission|denied/i)
              ? t(($) => $.denied)
              : t(($) => $.error)}
          </div>
        ) : tasks.length === 0 ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 rounded-lg border border-dashed p-10 text-sm text-muted-foreground">
            <CheckSquare2 className="size-8 opacity-40" />
            {t(($) => $.empty)}
          </div>
        ) : (
          <ul className="divide-y rounded-lg border bg-card">
            {tasks.map((task, index) => (
              <li key={task.id} className="flex items-center gap-3 p-3">
                <div className="flex flex-col">
                  <Button variant="ghost" size="icon-sm" disabled={index === 0 || reorderTasks.isPending} onClick={() => moveTask(index, -1)} aria-label={t(($) => $.move_up)}><ArrowUp className="size-3.5" /></Button>
                  <Button variant="ghost" size="icon-sm" disabled={index === tasks.length - 1 || reorderTasks.isPending} onClick={() => moveTask(index, 1)} aria-label={t(($) => $.move_down)}><ArrowDown className="size-3.5" /></Button>
                </div>
                <select
                  value={task.status}
                  disabled={task.status === "archived"}
                  onChange={(event) =>
                    updateTask.mutate({
                      id: task.id,
                      status: event.target.value as MutableTaskStatus,
                      expected_revision: task.revision,
                    })
                  }
                  className="h-8 rounded-md border bg-background px-2 text-xs"
                  aria-label={t(($) => $.status[task.status])}
                >
                  {NEXT_STATUSES[task.status].map((status) => (
                    <option key={status} value={status}>
                      {t(($) => $.status[status])}
                    </option>
                  ))}
                </select>
                <div className="min-w-0 flex-1">
                  <div className={task.status === "done" ? "truncate text-sm text-muted-foreground line-through" : "truncate text-sm"}>{task.title}</div>
                  <div className="mt-0.5 flex flex-wrap gap-x-3 text-xs text-muted-foreground">
                    {task.assignee_id ? <span>{task.assignee_type === "agent" ? t(($) => $.agent, { id: task.assignee_id }) : t(($) => $.member, { id: task.assignee_id })}</span> : null}
                    {task.due_date ? <span>{t(($) => $.due, { date: dueDateFormatter.format(new Date(task.due_date)) })}</span> : null}
                  </div>
                  {task.status !== "archived" ? (
                    <details key={`${task.id}:${task.revision}`} className="mt-2 text-xs">
                      <summary className="cursor-pointer text-muted-foreground">{t(($) => $.edit)}</summary>
                      <form
                        className="mt-2 grid gap-2 sm:grid-cols-2"
                        onSubmit={(event) => {
                          event.preventDefault();
                          const data = new FormData(event.currentTarget);
                          const assigneeId = String(data.get("assignee_id") ?? "").trim();
                          const assigneeType = String(data.get("assignee_type") ?? "") as TaskActorType;
                          const dueDate = String(data.get("due_date") ?? "").trim();
                          updateTask.mutate({
                            id: task.id,
                            title: String(data.get("title") ?? "").trim(),
                            due_date: dueDate || null,
                            assignee_type: assigneeId ? assigneeType : null,
                            assignee_id: assigneeId || null,
                            expected_revision: task.revision,
                          });
                        }}
                      >
                        <label className="grid gap-1">{t(($) => $.title_label)}<Input name="title" defaultValue={task.title} required /></label>
                        <label className="grid gap-1">{t(($) => $.due_label)}<Input name="due_date" type="date" defaultValue={task.due_date?.slice(0, 10) ?? ""} /></label>
                        <label className="grid gap-1">{t(($) => $.assignee_type_label)}
                          <select name="assignee_type" defaultValue={task.assignee_type ?? "member"} className="h-8 rounded-md border bg-background px-2">
                            <option value="member">{t(($) => $.assignee_member)}</option>
                            <option value="agent">{t(($) => $.assignee_agent)}</option>
                          </select>
                        </label>
                        <label className="grid gap-1">{t(($) => $.assignee_id_label)}<Input name="assignee_id" defaultValue={task.assignee_id ?? ""} /></label>
                        <Button type="submit" size="sm" className="sm:col-span-2" disabled={updateTask.isPending}>{t(($) => $.save)}</Button>
                      </form>
                    </details>
                  ) : null}
                </div>
                {task.status === "done" || task.status === "cancelled" ? (
                  <Button variant="ghost" size="icon-sm" onClick={() => deleteTask.mutate({ id: task.id, expectedRevision: task.revision })} aria-label={t(($) => $.archive)}>
                    <Archive className="size-4" />
                  </Button>
                ) : null}
                {task.status === "archived" ? (
                  <Button variant="ghost" size="icon-sm" onClick={() => restoreTask.mutate({ id: task.id, expectedRevision: task.revision })} aria-label={t(($) => $.restore)}>
                    <RotateCcw className="size-4" />
                  </Button>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  );
}
