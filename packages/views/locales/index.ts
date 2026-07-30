import type { LocaleResources, SupportedLocale } from "@multica/core/i18n";
import enCommon from "./en/common.json";
import enAuth from "./en/auth.json";
import enSettings from "./en/settings.json";
import enIssues from "./en/issues.json";
import enEditor from "./en/editor.json";
import enOnboarding from "./en/onboarding.json";
import enInvite from "./en/invite.json";
import enLabels from "./en/labels.json";
import enMembers from "./en/members.json";
import enMyIssues from "./en/my-issues.json";
import enSearch from "./en/search.json";
import enWorkspace from "./en/workspace.json";
import enProjects from "./en/projects.json";
import enTasks from "./en/tasks.json";
import enSkills from "./en/skills.json";
import enModals from "./en/modals.json";
import enLayout from "./en/layout.json";
import enUi from "./en/ui.json";
import zhHansCommon from "./zh-Hans/common.json";
import zhHansAuth from "./zh-Hans/auth.json";
import zhHansSettings from "./zh-Hans/settings.json";
import zhHansIssues from "./zh-Hans/issues.json";
import zhHansEditor from "./zh-Hans/editor.json";
import zhHansOnboarding from "./zh-Hans/onboarding.json";
import zhHansInvite from "./zh-Hans/invite.json";
import zhHansLabels from "./zh-Hans/labels.json";
import zhHansMembers from "./zh-Hans/members.json";
import zhHansMyIssues from "./zh-Hans/my-issues.json";
import zhHansSearch from "./zh-Hans/search.json";
import zhHansWorkspace from "./zh-Hans/workspace.json";
import zhHansProjects from "./zh-Hans/projects.json";
import zhHansTasks from "./zh-Hans/tasks.json";
import zhHansSkills from "./zh-Hans/skills.json";
import zhHansModals from "./zh-Hans/modals.json";
import zhHansLayout from "./zh-Hans/layout.json";
import zhHansUi from "./zh-Hans/ui.json";
import koCommon from "./ko/common.json";
import koAuth from "./ko/auth.json";
import koSettings from "./ko/settings.json";
import koIssues from "./ko/issues.json";
import koEditor from "./ko/editor.json";
import koOnboarding from "./ko/onboarding.json";
import koInvite from "./ko/invite.json";
import koLabels from "./ko/labels.json";
import koMembers from "./ko/members.json";
import koMyIssues from "./ko/my-issues.json";
import koSearch from "./ko/search.json";
import koWorkspace from "./ko/workspace.json";
import koProjects from "./ko/projects.json";
import koTasks from "./ko/tasks.json";
import koSkills from "./ko/skills.json";
import koModals from "./ko/modals.json";
import koLayout from "./ko/layout.json";
import koUi from "./ko/ui.json";
import jaCommon from "./ja/common.json";
import jaAuth from "./ja/auth.json";
import jaSettings from "./ja/settings.json";
import jaIssues from "./ja/issues.json";
import jaEditor from "./ja/editor.json";
import jaOnboarding from "./ja/onboarding.json";
import jaInvite from "./ja/invite.json";
import jaLabels from "./ja/labels.json";
import jaMembers from "./ja/members.json";
import jaMyIssues from "./ja/my-issues.json";
import jaSearch from "./ja/search.json";
import jaWorkspace from "./ja/workspace.json";
import jaProjects from "./ja/projects.json";
import jaTasks from "./ja/tasks.json";
import jaSkills from "./ja/skills.json";
import jaModals from "./ja/modals.json";
import jaLayout from "./ja/layout.json";
import jaUi from "./ja/ui.json";

// Single source of truth for the resource bundle. Both apps (web layout +
// desktop App.tsx) import from here so adding a locale or namespace happens
// in exactly one place.
export const RESOURCES: Record<SupportedLocale, LocaleResources> = {
  en: {
    common: enCommon,
    auth: enAuth,
    settings: enSettings,
    issues: enIssues,
    editor: enEditor,
    onboarding: enOnboarding,
    invite: enInvite,
    labels: enLabels,
    members: enMembers,
    "my-issues": enMyIssues,
    search: enSearch,
    workspace: enWorkspace,
    projects: enProjects,
    tasks: enTasks,
    skills: enSkills,
    modals: enModals,
    layout: enLayout,
    ui: enUi,
  },
  "zh-Hans": {
    common: zhHansCommon,
    auth: zhHansAuth,
    settings: zhHansSettings,
    issues: zhHansIssues,
    editor: zhHansEditor,
    onboarding: zhHansOnboarding,
    invite: zhHansInvite,
    labels: zhHansLabels,
    members: zhHansMembers,
    "my-issues": zhHansMyIssues,
    search: zhHansSearch,
    workspace: zhHansWorkspace,
    projects: zhHansProjects,
    tasks: zhHansTasks,
    skills: zhHansSkills,
    modals: zhHansModals,
    layout: zhHansLayout,
    ui: zhHansUi,
  },
  ko: {
    common: koCommon,
    auth: koAuth,
    settings: koSettings,
    issues: koIssues,
    editor: koEditor,
    onboarding: koOnboarding,
    invite: koInvite,
    labels: koLabels,
    members: koMembers,
    "my-issues": koMyIssues,
    search: koSearch,
    workspace: koWorkspace,
    projects: koProjects,
    tasks: koTasks,
    skills: koSkills,
    modals: koModals,
    layout: koLayout,
    ui: koUi,
  },
  ja: {
    common: jaCommon,
    auth: jaAuth,
    settings: jaSettings,
    issues: jaIssues,
    editor: jaEditor,
    onboarding: jaOnboarding,
    invite: jaInvite,
    labels: jaLabels,
    members: jaMembers,
    "my-issues": jaMyIssues,
    search: jaSearch,
    workspace: jaWorkspace,
    projects: jaProjects,
    tasks: jaTasks,
    skills: jaSkills,
    modals: jaModals,
    layout: jaLayout,
    ui: jaUi,
  },
};
