export type TaskStatus = "todo" | "in_progress" | "done" | "cancelled" | "archived";
export type TaskTerminalStatus = "done" | "cancelled";
export type TaskPriority = "urgent" | "high" | "medium" | "low" | "none";
export type TaskActorType = "member" | "agent";

export interface Task {
  id: string;
  workspace_id: string;
  project_id: string | null;
  issue_id: string | null;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  assignee_type: TaskActorType | null;
  assignee_id: string | null;
  creator_type: TaskActorType;
  creator_id: string;
  position: number;
  revision: number;
  start_date: string | null;
  due_date: string | null;
  completed_at: string | null;
  archived_at: string | null;
  restore_status: TaskTerminalStatus | "";
  created_at: string;
  updated_at: string;
}

export interface CreateTaskRequest {
  project_id?: string | null;
  issue_id?: string | null;
  title: string;
  description?: string;
  status?: Exclude<TaskStatus, "archived">;
  priority?: TaskPriority;
  assignee_type?: TaskActorType | null;
  assignee_id?: string | null;
  position?: number;
  start_date?: string | null;
  due_date?: string | null;
}

export interface UpdateTaskRequest extends Partial<Omit<CreateTaskRequest, "title">> {
  title?: string;
  expected_revision: number;
}

export interface TaskRevisionRequest {
  expected_revision: number;
}

export interface ReorderTaskItem extends TaskRevisionRequest {
  id: string;
  position: number;
}

export interface ReorderTasksRequest {
  items: ReorderTaskItem[];
}

export interface ListTasksResponse {
  tasks: Task[];
  total: number;
}
