import type { LocaleResources, SupportedLocale } from "@multica/core/i18n";
import enCommon from "./en/common.json";
import enSettings from "./en/settings.json";
import enIssues from "./en/issues.json";
import enAgents from "./en/agents.json";
import enEditor from "./en/editor.json";
import enLabels from "./en/labels.json";
import enMembers from "./en/members.json";
import enMyIssues from "./en/my-issues.json";
import enSearch from "./en/search.json";
import enInbox from "./en/inbox.json";
import enWorkspace from "./en/workspace.json";
import enFeatures from "./en/features.json";
import enAutopilots from "./en/autopilots.json";
import enSkills from "./en/skills.json";
import enModals from "./en/modals.json";
import enRuntimes from "./en/runtimes.json";
import enLayout from "./en/layout.json";
import enUsage from "./en/usage.json";
import enUi from "./en/ui.json";
// Single source of truth for the resource bundle.
export const RESOURCES: Record<SupportedLocale, LocaleResources> = {
  en: {
    common: enCommon,
    settings: enSettings,
    issues: enIssues,
    agents: enAgents,
    editor: enEditor,
    labels: enLabels,
    members: enMembers,
    "my-issues": enMyIssues,
    search: enSearch,
    inbox: enInbox,
    workspace: enWorkspace,
    features: enFeatures,
    autopilots: enAutopilots,
    skills: enSkills,
    modals: enModals,
    runtimes: enRuntimes,
    layout: enLayout,
    usage: enUsage,
    ui: enUi,
  },
};
