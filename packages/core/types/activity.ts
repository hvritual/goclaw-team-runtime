import type { Attachment } from "./attachment";
import type { Reaction } from "./comment";

export interface TimelineEntry {
  type: string;
  id: string;
  actor_type: string;
  actor_id: string;
  created_at: string;
  action?: string;
  details?: Record<string, unknown>;
  content?: string;
  parent_id?: string | null;
  updated_at?: string;
  comment_type?: string;
  reactions?: Reaction[];
  attachments?: Attachment[];
  resolved_at?: string | null;
  resolved_by_type?: string | null;
  resolved_by_id?: string | null;
  coalesced_count?: number;
}
