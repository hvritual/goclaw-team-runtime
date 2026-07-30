import React from "react";
import { vi } from "vitest";
import { render, type RenderOptions } from "@testing-library/react";
import type { User, Workspace, MemberWithUser } from "@multica/core/types";

// Mock user
export const mockUser: User = {
  id: "user-1",
  name: "Test User",
  email: "test@multica.ai",
  avatar_url: null,
  onboarded_at: "2026-01-01T00:00:00Z",
  onboarding_questionnaire: {},
  // Matches real server behavior for anyone who onboarded before this
  // field shipped — migration 054 backfills 'skipped_legacy'.
  starter_content_state: "skipped_legacy",
  language: null,
  timezone: null,
  profile_description: "",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

// Mock workspace
export const mockWorkspace: Workspace = {
  id: "ws-1",
  name: "Test Workspace",
  slug: "test-ws",
  description: "A test workspace",
  context: null,
  settings: {},
  repos: [],
  issue_prefix: "TES",
  avatar_url: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

// Mock members
export const mockMembers: MemberWithUser[] = [
  {
    id: "member-1",
    workspace_id: "ws-1",
    user_id: "user-1",
    role: "owner",
    created_at: "2026-01-01T00:00:00Z",
    name: "Test User",
    email: "test@multica.ai",
    avatar_url: null,
  },
];

// Mock auth context value
 
export const mockAuthValue: Record<string, any> = {
  user: mockUser,
  workspace: mockWorkspace,
  members: mockMembers,
  isLoading: false,
  login: vi.fn(),
  logout: vi.fn(),
  updateWorkspace: vi.fn(),
  updateCurrentUser: vi.fn(),
  getMemberName: (userId: string) => {
    const m = mockMembers.find((m) => m.user_id === userId);
    return m?.name ?? "Unknown";
  },
  getActorName: (type: string, id: string) => {
    if (type === "member") {
      const m = mockMembers.find((m) => m.user_id === id);
      return m?.name ?? "Unknown";
    }
    return "System";
  },
  getActorInitials: (type: string, id: string) => {
    return "TU";
  },
};
