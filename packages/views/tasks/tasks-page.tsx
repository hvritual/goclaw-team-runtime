"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { CheckSquare2, Loader2, Plus, Trash2 } from "lucide-react";
import type { TaskStatus } from "@multica/core/types";
import { useWorkspaceId } from "@multica/core/hooks";
import {
  taskListOptions,
  useCreateTask,
  useDeleteTask,
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
];

export function TasksPage() {
  const { t } = useT("tasks");
  const workspaceId = useWorkspaceId();
  const { data: tasks = [], isLoading } = useQuery(
    taskListOptions(workspaceId),
  );
  const createTask = useCreateTask();
  const updateTask = useUpdateTask();
  const deleteTask = useDeleteTask();
  const [title, setTitle] = useState("");

  const submit = () => {
    const value = title.trim();
    if (!value) return;
    createTask.mutate(
      { title: value },
      { onSuccess: () => setTitle("") },
    );
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
        <CheckSquare2 className="size-4 text-muted-foreground" />
        <h1 className="text-sm font-semibold">{t(($) => $.title)}</h1>
      </header>

      <main className="mx-auto flex w-full max-w-4xl flex-1 flex-col gap-4 overflow-y-auto p-4 sm:p-6">
        <div className="flex gap-2">
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
        </div>

        {isLoading ? (
          <div className="flex flex-1 items-center justify-center">
            <Loader2 className="size-5 animate-spin text-muted-foreground" />
          </div>
        ) : tasks.length === 0 ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 rounded-lg border border-dashed p-10 text-sm text-muted-foreground">
            <CheckSquare2 className="size-8 opacity-40" />
            {t(($) => $.empty)}
          </div>
        ) : (
          <ul className="divide-y rounded-lg border bg-card">
            {tasks.map((task) => (
              <li key={task.id} className="flex items-center gap-3 p-3">
                <select
                  value={task.status}
                  onChange={(event) =>
                    updateTask.mutate({
                      id: task.id,
                      status: event.target.value as TaskStatus,
                    })
                  }
                  className="h-8 rounded-md border bg-background px-2 text-xs"
                  aria-label={t(($) => $.status[task.status])}
                >
                  {STATUSES.map((status) => (
                    <option key={status} value={status}>
                      {t(($) => $.status[status])}
                    </option>
                  ))}
                </select>
                <span
                  className={
                    task.status === "done"
                      ? "min-w-0 flex-1 truncate text-sm text-muted-foreground line-through"
                      : "min-w-0 flex-1 truncate text-sm"
                  }
                >
                  {task.title}
                </span>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => deleteTask.mutate(task.id)}
                  aria-label={t(($) => $.delete)}
                >
                  <Trash2 className="size-4" />
                </Button>
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  );
}
