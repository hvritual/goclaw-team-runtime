export interface IssueSubscriber {
  issue_id: string;
  user_type: "member";
  user_id: string;
  reason: "creator" | "assignee" | "commenter" | "mentioned" | "manual";
  created_at: string;
}
