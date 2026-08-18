export interface SkillSummary {
  id: string;
  workspace_id: string;
  version_id: string;
  version: string;
  name: string;
  description: string;
  config: Record<string, unknown>;
  status: "draft" | "published" | "deprecated" | "archived";
  revision: number;
  created_by: string;
  created_at: string;
  updated_at: string;
  archived: boolean;
}

export interface Skill extends SkillSummary {
  content: string;
  files: SkillFile[];
}

export interface SkillFile {
  id: string;
  skill_id: string;
  path: string;
  content: string;
  created_at: string;
  updated_at: string;
}

export interface SkillHistory {
  skill_id: string;
  provenance: {
    origin_workspace_id: string;
    created_by: string;
    created_at: string;
  };
  audit: {
    id: string;
    version_id: string;
    workspace_id: string;
    actor_type: string;
    actor_id: string;
    action: string;
    created_at: string;
  }[];
}

export interface CreateSkillRequest {
  name: string;
  description?: string;
  content?: string;
  config?: Record<string, unknown>;
  files?: { path: string; content: string }[];
}

export interface UpdateSkillRequest {
  name?: string;
  description?: string;
  content?: string;
  config?: Record<string, unknown>;
  files?: { path: string; content: string }[];
  expected_revision?: number;
}
