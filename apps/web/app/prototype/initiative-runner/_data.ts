// Mock data for the initiative-runner prototype. Throwaway — delete with
// the prototype once a variant is chosen.

export type Status = "ready" | "running" | "in_review" | "done" | "blocked";

export type Initiative = {
  id: string;
  title: string;
  prd: string;
  status: Status;
  mode: "worktree" | "in_place";
  milestonesTotal: number;
  milestonesPassed: number;
  issuesTotal: number;
  issuesDone: number;
  startedAt: string;
  lastActivityAt: string;
  branchSlug: string;
  prUrl?: string;
};

export type Agent = {
  id: string;
  name: string;
  backend: "claude" | "codex" | "gemini";
  hue: number; // 0..360
};

export type Phase =
  | "thinking"
  | "editing"
  | "running_tests"
  | "committing"
  | "waiting_local_dir";

export type IssueStatus =
  | "todo"
  | "in_progress"
  | "in_review"
  | "done"
  | "blocked";

export type Issue = {
  id: string;
  initiativeId: string;
  number: number;
  title: string;
  status: IssueStatus;
  assigneeId?: string;
  phase?: Phase;
  heartbeatMs?: number; // ms since last heartbeat
  toolCount?: number;
  editCount?: number;
  branchSlug: string;
  repo: string;
  milestone: string;
};

export type ActivityType =
  | "agent_started"
  | "tool_use"
  | "edit"
  | "commit"
  | "milestone_passed"
  | "milestone_failed"
  | "issue_done"
  | "initiative_ready_for_review"
  | "dod_failed"
  | "tripwire_paused";

export type ActivityEvent = {
  id: string;
  tsMinutesAgo: number;
  type: ActivityType;
  initiativeId: string;
  issueId?: string;
  agentId?: string;
  message: string;
};

export type DecisionEntry = {
  id: string;
  initiativeId: string;
  tsMinutesAgo: number;
  title: string;
  decision: string;
  learning: string;
  adrRefs?: string[];
  contextTerms?: string[];
};

export const AGENTS: Agent[] = [
  { id: "a1", name: "claude-sonnet", backend: "claude", hue: 268 },
  { id: "a2", name: "claude-opus", backend: "claude", hue: 24 },
  { id: "a3", name: "codex-medium", backend: "codex", hue: 142 },
  { id: "a4", name: "gemini-pro", backend: "gemini", hue: 198 },
  { id: "a5", name: "claude-haiku", backend: "claude", hue: 310 },
];

export const INITIATIVES: Initiative[] = [
  {
    id: "i1",
    title: "Refatorar checkout pra Stripe Elements",
    prd: "Migrar o fluxo de checkout de Charge API para Stripe Elements + PaymentIntents. Inclui 3D Secure, salvamento de cartões via SetupIntent, e webhook de confirmação. DoD: testes E2E passando em cartão BR/US, métrica de conversão preservada em ±2pp.",
    status: "running",
    mode: "worktree",
    milestonesTotal: 4,
    milestonesPassed: 2,
    issuesTotal: 8,
    issuesDone: 5,
    startedAt: "2026-06-01T14:00:00Z",
    lastActivityAt: "12s ago",
    branchSlug: "checkout-stripe-elements",
  },
  {
    id: "i2",
    title: "Migrar feature flags pra OpenFeature",
    prd: "Substituir LaunchDarkly por implementação self-hosted baseada em OpenFeature SDK. Manter compatibilidade de signatures durante 2 sprints. DoD: 100% das flags migradas, 0 regressões em A/B tests em curso, dashboard interno funcional.",
    status: "in_review",
    mode: "worktree",
    milestonesTotal: 3,
    milestonesPassed: 3,
    issuesTotal: 12,
    issuesDone: 12,
    startedAt: "2026-05-28T10:00:00Z",
    lastActivityAt: "4min ago",
    branchSlug: "openfeature-migration",
    prUrl: "https://github.com/example/repo/pull/4521",
  },
  {
    id: "i3",
    title: "Auditoria de N+1 queries no perfil do usuário",
    prd: "Identificar e corrigir N+1 queries nas rotas de perfil que estão pagando 90% do P95. DoD: P95 < 200ms em /me e /me/orders, total de queries por request <= 8.",
    status: "running",
    mode: "in_place",
    milestonesTotal: 2,
    milestonesPassed: 0,
    issuesTotal: 5,
    issuesDone: 1,
    startedAt: "2026-06-03T09:00:00Z",
    lastActivityAt: "just now",
    branchSlug: "user-profile-n-plus-one",
  },
  {
    id: "i4",
    title: "Localização pt-BR no painel admin",
    prd: "Extrair strings hardcoded do painel admin e wrappar com Lingui. Cobertura mínima 95% das strings visíveis. DoD: snapshot tests verdes em pt-BR e en-US, sem strings hardcoded em /admin/**.",
    status: "blocked",
    mode: "worktree",
    milestonesTotal: 2,
    milestonesPassed: 0,
    issuesTotal: 6,
    issuesDone: 2,
    startedAt: "2026-06-02T08:30:00Z",
    lastActivityAt: "2h ago",
    branchSlug: "admin-pt-br-i18n",
  },
  {
    id: "i5",
    title: "Remover legacy Redis cache no search",
    prd: "Substituir o Redis cache de busca (descontinuado) por cache em memória com TTL ajustável e invalidação por evento. DoD: P95 busca dentro de 5% do baseline atual, ZERO calls a Redis no path quente.",
    status: "done",
    mode: "worktree",
    milestonesTotal: 2,
    milestonesPassed: 2,
    issuesTotal: 4,
    issuesDone: 4,
    startedAt: "2026-05-20T11:00:00Z",
    lastActivityAt: "merged 8d ago",
    branchSlug: "search-no-redis",
    prUrl: "https://github.com/example/repo/pull/4498",
  },
];

export const ISSUES: Issue[] = [
  // i1 (running, mid-execution)
  {
    id: "is1",
    initiativeId: "i1",
    number: 312,
    title: "Setup PaymentIntent endpoint",
    status: "done",
    branchSlug: "checkout-stripe-elements",
    repo: "api-checkout",
    milestone: "M1: Backend ready",
  },
  {
    id: "is2",
    initiativeId: "i1",
    number: 313,
    title: "SetupIntent flow for saved cards",
    status: "done",
    branchSlug: "checkout-stripe-elements",
    repo: "api-checkout",
    milestone: "M1: Backend ready",
  },
  {
    id: "is3",
    initiativeId: "i1",
    number: 314,
    title: "Webhook handler with idempotency",
    status: "done",
    branchSlug: "checkout-stripe-elements",
    repo: "api-checkout",
    milestone: "M2: Webhooks",
  },
  {
    id: "is4",
    initiativeId: "i1",
    number: 315,
    title: "Frontend Elements integration",
    status: "in_progress",
    assigneeId: "a1",
    phase: "editing",
    heartbeatMs: 1200,
    toolCount: 18,
    editCount: 7,
    branchSlug: "checkout-stripe-elements",
    repo: "web-checkout",
    milestone: "M3: Frontend",
  },
  {
    id: "is5",
    initiativeId: "i1",
    number: 316,
    title: "3DS challenge UX",
    status: "in_progress",
    assigneeId: "a3",
    phase: "running_tests",
    heartbeatMs: 800,
    toolCount: 22,
    editCount: 11,
    branchSlug: "checkout-stripe-elements",
    repo: "web-checkout",
    milestone: "M3: Frontend",
  },
  {
    id: "is6",
    initiativeId: "i1",
    number: 317,
    title: "E2E tests for BR/US cards",
    status: "todo",
    branchSlug: "checkout-stripe-elements",
    repo: "web-checkout",
    milestone: "M4: E2E",
  },
  {
    id: "is7",
    initiativeId: "i1",
    number: 318,
    title: "Conversion-rate dashboard probe",
    status: "todo",
    branchSlug: "checkout-stripe-elements",
    repo: "ops-dashboards",
    milestone: "M4: E2E",
  },
  {
    id: "is8",
    initiativeId: "i1",
    number: 319,
    title: "Migration plan for stored payment methods",
    status: "done",
    branchSlug: "checkout-stripe-elements",
    repo: "api-checkout",
    milestone: "M1: Backend ready",
  },
  // i2 (in_review)
  {
    id: "is9",
    initiativeId: "i2",
    number: 287,
    title: "OpenFeature provider scaffold",
    status: "done",
    branchSlug: "openfeature-migration",
    repo: "platform-flags",
    milestone: "M1: Provider",
  },
  {
    id: "is10",
    initiativeId: "i2",
    number: 288,
    title: "Migrate boolean flags (47)",
    status: "done",
    branchSlug: "openfeature-migration",
    repo: "platform-flags",
    milestone: "M2: Mass migration",
  },
  {
    id: "is11",
    initiativeId: "i2",
    number: 289,
    title: "Multivariate flag migration (12)",
    status: "done",
    branchSlug: "openfeature-migration",
    repo: "platform-flags",
    milestone: "M2: Mass migration",
  },
  // i3 (running, just started)
  {
    id: "is12",
    initiativeId: "i3",
    number: 401,
    title: "Profile route N+1 audit",
    status: "in_progress",
    assigneeId: "a2",
    phase: "thinking",
    heartbeatMs: 400,
    toolCount: 4,
    editCount: 0,
    branchSlug: "user-profile-n-plus-one",
    repo: "api-profile",
    milestone: "M1: Discovery",
  },
  {
    id: "is13",
    initiativeId: "i3",
    number: 402,
    title: "Orders route N+1 audit",
    status: "in_progress",
    assigneeId: "a4",
    phase: "waiting_local_dir",
    heartbeatMs: 60000,
    toolCount: 1,
    editCount: 0,
    branchSlug: "user-profile-n-plus-one",
    repo: "api-orders",
    milestone: "M1: Discovery",
  },
  {
    id: "is14",
    initiativeId: "i3",
    number: 403,
    title: "Add DataLoader to GraphQL resolvers",
    status: "todo",
    branchSlug: "user-profile-n-plus-one",
    repo: "api-profile",
    milestone: "M2: Fix",
  },
  // i4 (blocked)
  {
    id: "is15",
    initiativeId: "i4",
    number: 220,
    title: "Extract strings — Settings tab",
    status: "blocked",
    branchSlug: "admin-pt-br-i18n",
    repo: "web-admin",
    milestone: "M1: Strings",
  },
];

export const ACTIVITY: ActivityEvent[] = [
  { id: "e1", tsMinutesAgo: 0, type: "tool_use", initiativeId: "i3", issueId: "is12", agentId: "a2", message: "ran read_file: api-profile/src/resolvers/me.ts" },
  { id: "e2", tsMinutesAgo: 0, type: "edit", initiativeId: "i1", issueId: "is4", agentId: "a1", message: "edited web-checkout/components/PaymentForm.tsx (+42 -18)" },
  { id: "e3", tsMinutesAgo: 1, type: "tool_use", initiativeId: "i1", issueId: "is5", agentId: "a3", message: "ran pnpm test --filter=web-checkout" },
  { id: "e4", tsMinutesAgo: 3, type: "agent_started", initiativeId: "i3", issueId: "is13", agentId: "a4", message: "claimed 402 — waiting on local directory lock" },
  { id: "e5", tsMinutesAgo: 4, type: "commit", initiativeId: "i2", issueId: "is11", agentId: "a1", message: "fix: multivariate fallback when provider returns null" },
  { id: "e6", tsMinutesAgo: 8, type: "milestone_passed", initiativeId: "i2", message: "M2: Mass migration — DoD green (all assertions passed)" },
  { id: "e7", tsMinutesAgo: 12, type: "issue_done", initiativeId: "i1", issueId: "is3", agentId: "a1", message: "Webhook handler with idempotency — done" },
  { id: "e8", tsMinutesAgo: 18, type: "initiative_ready_for_review", initiativeId: "i2", message: "Initiative ready for human review — PR #4521 opened" },
  { id: "e9", tsMinutesAgo: 22, type: "dod_failed", initiativeId: "i1", message: "M2: Webhooks — assertion 'idempotency under 100 concurrent' failed; auto-creating follow-up" },
  { id: "e10", tsMinutesAgo: 35, type: "edit", initiativeId: "i1", issueId: "is5", agentId: "a3", message: "edited web-checkout/components/ThreeDS.tsx (+88 -2)" },
  { id: "e11", tsMinutesAgo: 41, type: "tripwire_paused", initiativeId: "i4", message: "Tripwire fired: failure tolerance (3 same-Milestone failures) — Initiative paused" },
  { id: "e12", tsMinutesAgo: 55, type: "tool_use", initiativeId: "i1", issueId: "is4", agentId: "a1", message: "ran grep -rn 'Stripe.handleCardPayment' web-checkout/" },
  { id: "e13", tsMinutesAgo: 64, type: "milestone_passed", initiativeId: "i1", message: "M2: Webhooks — DoD green (re-validated after follow-up)" },
  { id: "e14", tsMinutesAgo: 88, type: "agent_started", initiativeId: "i1", issueId: "is4", agentId: "a1", message: "claimed 315 — Frontend Elements integration" },
  { id: "e15", tsMinutesAgo: 92, type: "commit", initiativeId: "i1", issueId: "is3", agentId: "a1", message: "feat: idempotency key check on webhook reception" },
  { id: "e16", tsMinutesAgo: 120, type: "milestone_failed", initiativeId: "i1", message: "M2: Webhooks — 1/3 assertions failed; follow-up issue created" },
  { id: "e17", tsMinutesAgo: 180, type: "issue_done", initiativeId: "i2", issueId: "is11", agentId: "a1", message: "Multivariate flag migration (12) — done" },
];

export const DECISIONS: DecisionEntry[] = [
  {
    id: "d1",
    initiativeId: "i1",
    tsMinutesAgo: 22,
    title: "Webhook idempotency: stripe-signature header alone is not enough",
    decision: "Use signature header for authenticity + event id for dedup; persist event ids in redis with 24h TTL.",
    learning: "Stripe retries events for up to 24h on 5xx — a 200 without dedup creates double-billings under intermittent failures.",
    adrRefs: ["ADR-0012-payment-webhook-idempotency"],
    contextTerms: ["PaymentWebhook", "IdempotencyKey"],
  },
  {
    id: "d2",
    initiativeId: "i1",
    tsMinutesAgo: 64,
    title: "Use Elements over Checkout: customisation > convenience",
    decision: "Stripe Elements (custom UI) over Checkout (hosted) — we need the 3DS UX integrated in our flow, not as a redirect.",
    learning: "Hosted Checkout is faster to ship but conversion drops ~6pp on mobile; Elements wins long-term despite higher upfront cost.",
    adrRefs: ["ADR-0011-stripe-elements"],
    contextTerms: ["PaymentForm", "3DSChallenge"],
  },
  {
    id: "d3",
    initiativeId: "i2",
    tsMinutesAgo: 180,
    title: "Migrate flag-by-flag, not module-by-module",
    decision: "Iterate over every flag (boolean first, multivariate second) instead of per-module batches.",
    learning: "Per-module created cross-module flag mismatches during the window; per-flag is slower per PR but the surface stays consistent.",
    contextTerms: ["FeatureFlag", "OpenFeatureProvider"],
  },
  {
    id: "d4",
    initiativeId: "i4",
    tsMinutesAgo: 41,
    title: "Admin tabs use untranslated date formats — postpone",
    decision: "Skip date strings in this initiative; carve out a follow-up Initiative for date locale handling.",
    learning: "Lingui-wrapping date strings before the formatter migration produces double-formatted output.",
    contextTerms: ["DateFormatter"],
  },
];

export const agentById = (id?: string) => (id ? AGENTS.find((a) => a.id === id) : undefined);
export const initiativeById = (id: string) => INITIATIVES.find((i) => i.id === id);
export const issueById = (id: string) => ISSUES.find((i) => i.id === id);
export const issuesByInitiative = (id: string) => ISSUES.filter((i) => i.initiativeId === id);
export const activityByInitiative = (id: string) => ACTIVITY.filter((a) => a.initiativeId === id);
export const decisionsByInitiative = (id: string) => DECISIONS.filter((d) => d.initiativeId === id);

export const STATUS_LABEL: Record<Status, string> = {
  ready: "Ready",
  running: "Running",
  in_review: "In Review",
  done: "Done",
  blocked: "Blocked",
};

export const PHASE_LABEL: Record<Phase, string> = {
  thinking: "Thinking",
  editing: "Editing",
  running_tests: "Running tests",
  committing: "Committing",
  waiting_local_dir: "Waiting on dir",
};

export const ISSUE_STATUS_LABEL: Record<IssueStatus, string> = {
  todo: "Todo",
  in_progress: "In progress",
  in_review: "In review",
  done: "Done",
  blocked: "Blocked",
};

export const formatActivityType = (t: ActivityType): string => {
  switch (t) {
    case "agent_started": return "started";
    case "tool_use": return "tool";
    case "edit": return "edit";
    case "commit": return "commit";
    case "milestone_passed": return "milestone";
    case "milestone_failed": return "DoD fail";
    case "issue_done": return "issue done";
    case "initiative_ready_for_review": return "ready for review";
    case "dod_failed": return "DoD fail";
    case "tripwire_paused": return "tripwire";
  }
};

export const formatTsAgo = (m: number): string => {
  if (m === 0) return "just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
};
