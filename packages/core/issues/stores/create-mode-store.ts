"use client";

import { useModalStore } from "../../modals";

export function openCreateIssueWithPreference(
  data?: Record<string, unknown> | null,
) {
  useModalStore.getState().open("create-issue", data ?? null);
}
