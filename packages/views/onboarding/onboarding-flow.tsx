"use client";

import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { setCurrentWorkspace } from "@multica/core/platform";
import { completeOnboarding } from "@multica/core/onboarding";
import { workspaceListOptions } from "@multica/core/workspace/queries";
import type { Workspace } from "@multica/core/types";
import { StepWorkspace } from "./steps/step-workspace";
import { useT } from "../i18n";

export type OnboardingStep = "workspace";

export function OnboardingFlow({
  onComplete,
}: {
  onComplete: (workspace?: Workspace) => void;
}) {
  const { t } = useT("onboarding");
  const { data: workspaces = [] } = useQuery(workspaceListOptions());

  const finish = async (workspace: Workspace) => {
    try {
      await completeOnboarding("full", workspace.id);
      setCurrentWorkspace(workspace.slug, workspace.id);
      onComplete(workspace);
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t(($) => $.errors.skip_failed),
      );
    }
  };

  return (
    <StepWorkspace
      existing={workspaces[0] ?? null}
      onCreated={finish}
    />
  );
}
