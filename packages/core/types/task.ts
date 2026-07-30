export type TaskStatus = "todo" | "in_progress" | "done" | "cancelled";
export type TaskPriority = "urgent" | "high" | "medium" | "low" | "none";

export interface Task {
  id: string;
  workspace_id: string;
  project_id: string | null;
  issue_id: string | null;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  assignee_id: string | null;
  creator_id: string;
  position: number;
  start_date: string | null;
  due_date: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateTaskRequest {
  project_id?: string | null;
  issue_id?: string | null;
  title: string;
  description?: string;
  status?: TaskStatus;
  priority?: TaskPriority;
  assignee_id?: string | null;
  position?: number;
  start_date?: string | null;
  due_date?: string | null;
}

export type UpdateTaskRequest = Partial<CreateTaskRequest>;

export interface ListTasksResponse {
  tasks: Task[];
  total: number;
}
