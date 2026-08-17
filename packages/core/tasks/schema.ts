import { z } from "zod";
import type { ListTasksResponse } from "../types/task";

const nullableString = z.string().nullable();

export const taskSchema = z.object({
  id: z.string().min(1),
  workspace_id: z.string().min(1),
  project_id: nullableString,
  issue_id: nullableString,
  title: z.string().min(1),
  description: z.string(),
  status: z.enum(["todo", "in_progress", "done", "cancelled", "archived"]),
  priority: z.enum(["urgent", "high", "medium", "low", "none"]),
  assignee_type: z.enum(["member", "agent"]).nullable(),
  assignee_id: nullableString,
  creator_type: z.enum(["member", "agent"]),
  creator_id: z.string().min(1),
  position: z.number().finite(),
  revision: z.number().int().positive(),
  start_date: nullableString,
  due_date: nullableString,
  completed_at: nullableString,
  archived_at: nullableString,
  restore_status: z.union([z.literal(""), z.literal("done"), z.literal("cancelled")]),
  created_at: z.string().min(1),
  updated_at: z.string().min(1),
}).strict().superRefine((task, context) => {
  if ((task.assignee_type === null) !== (task.assignee_id === null)) {
    context.addIssue({ code: "custom", message: "task assignee fields must be paired", path: ["assignee_id"] });
  }
});

export const taskListSchema = z.object({
  tasks: z.array(taskSchema),
  total: z.number().int().nonnegative(),
}).strict();

export const reorderedTasksSchema = z.object({ tasks: z.array(taskSchema) }).strict();

export const EMPTY_TASK_LIST: ListTasksResponse = { tasks: [], total: 0 };
