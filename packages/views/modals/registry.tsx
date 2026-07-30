"use client";

import { useModalStore } from "@multica/core/modals";
import { CreateWorkspaceModal } from "./create-workspace";
import { CreateIssueDialog } from "./create-issue-dialog";
import { CreateProjectModal } from "./create-project";
import { FeedbackModal } from "./feedback";
import { SetParentIssueModal } from "./set-parent-issue";
import { AddChildIssueModal } from "./add-child-issue";
import { DeleteIssueConfirmModal } from "./delete-issue-confirm";

export function ModalRegistry() {
  const modal = useModalStore((s) => s.modal);
  const data = useModalStore((s) => s.data);
  const close = useModalStore((s) => s.close);

  switch (modal) {
    case "create-workspace":
      return <CreateWorkspaceModal onClose={close} />;
    case "create-issue":
      return <CreateIssueDialog onClose={close} data={data} />;
    case "create-project":
      return <CreateProjectModal onClose={close} />;
    case "feedback":
      return <FeedbackModal onClose={close} data={data} />;
    case "issue-set-parent":
      return <SetParentIssueModal onClose={close} data={data} />;
    case "issue-add-child":
      return <AddChildIssueModal onClose={close} data={data} />;
    case "issue-delete-confirm":
      return <DeleteIssueConfirmModal onClose={close} data={data} />;
    default:
      return null;
  }
}
