"use client";

import { useState } from "react";
import { Dialog, DialogContent } from "@multica/ui/components/ui/dialog";
import { ManualCreatePanel, manualDialogContentClass } from "./create-issue";

export function CreateIssueDialog({
  onClose,
  data,
}: {
  onClose: () => void;
  data?: Record<string, unknown> | null;
}) {
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent
        finalFocus={false}
        showCloseButton={false}
        className={manualDialogContentClass(isExpanded)}
      >
        <ManualCreatePanel
          onClose={onClose}
          data={data}
          isExpanded={isExpanded}
          setIsExpanded={setIsExpanded}
        />
      </DialogContent>
    </Dialog>
  );
}
