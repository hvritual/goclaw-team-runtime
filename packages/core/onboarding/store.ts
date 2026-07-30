import { api } from "../api";
import { useAuthStore } from "../auth";
import type { OnboardingCompletionPath } from "./types";

/**
 * Finalize onboarding. POST /complete marks `onboarded_at` atomically
 * (COALESCE-guarded for idempotency) and emits the `onboarding_completed`
 * analytics event exactly once. We then refresh the auth store so every
 * gate sees the updated user — most importantly the workspace layout
 * hard gate that redirects un-onboarded users back to /onboarding.
 * `completionPath` is the client's view of how the user completed onboarding;
 * took; the server funnel-splits `onboarding_completed` on this value.
 * Legacy callers that don't pass a path get recorded as `unknown`.
 */
export async function completeOnboarding(
  completionPath?: OnboardingCompletionPath,
  workspaceId?: string,
): Promise<void> {
  await api.markOnboardingComplete(
    completionPath || workspaceId
      ? { completion_path: completionPath, workspace_id: workspaceId }
      : undefined,
  );
  await useAuthStore.getState().refreshMe();
}
