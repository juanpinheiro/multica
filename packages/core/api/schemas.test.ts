import { describe, expect, it } from "vitest";
import {
  DashboardAgentRunTimeListSchema,
  DashboardUsageByAgentListSchema,
  DashboardUsageDailyListSchema,
  DecisionLogEntrySchema,
  DodAssertionSchema,
  DuplicateIssueErrorBodySchema,
  EMPTY_DOD_ASSERTION,
  EMPTY_LIST_DECISION_LOG_RESPONSE,
  EMPTY_FEATURE,
  EMPTY_FEATURE_ISSUES_RESPONSE,
  EMPTY_LIST_DOD_ASSERTIONS_RESPONSE,
  EMPTY_LIST_HANDOFFS_RESPONSE,
  EMPTY_LIST_MILESTONES_RESPONSE,
  EMPTY_MILESTONE,
  EMPTY_USER,
  FeatureIssuesResponseSchema,
  FeatureSchema,
  HandoffSchema,
  ListDecisionLogResponseSchema,
  ListDodAssertionsResponseSchema,
  ListHandoffsResponseSchema,
  ListIssuesResponseSchema,
  ListMilestonesResponseSchema,
  MilestoneSchema,
  RuntimeHourlyActivityListSchema,
  RuntimeUsageByAgentListSchema,
  RuntimeUsageByHourListSchema,
  RuntimeUsageListSchema,
  UserSchema,
} from "./schemas";
import { parseWithFallback } from "./schema";

const baseIssue = {
  id: "11111111-1111-1111-1111-111111111111",
  workspace_id: "ws-1",
  number: 1,
  identifier: "MUL-1",
  title: "Test",
  description: null,
  status: "todo",
  priority: "medium",
  assignee_type: null,
  assignee_id: null,
  creator_type: "member",
  creator_id: "user-1",
  parent_issue_id: null,
  feature_id: null,
  position: 0,
  start_date: null,
  due_date: null,
  metadata: {},
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("IssueSchema (via ListIssuesResponseSchema)", () => {
  it("accepts a primitive metadata KV map", () => {
    const payload = {
      issues: [
        {
          ...baseIssue,
          metadata: { pipeline_status: "waiting", pr_number: 3, is_blocked: true },
        },
      ],
      total: 1,
    };
    const parsed = ListIssuesResponseSchema.parse(payload);
    expect(parsed.issues[0]?.metadata).toEqual({
      pipeline_status: "waiting",
      pr_number: 3,
      is_blocked: true,
    });
  });

  it("defaults metadata to {} when the server omits it (older backend)", () => {
    const { metadata: _omit, ...issueWithoutMetadata } = baseIssue;
    const payload = { issues: [issueWithoutMetadata], total: 1 };
    const parsed = ListIssuesResponseSchema.parse(payload);
    expect(parsed.issues[0]?.metadata).toEqual({});
  });

  it("rejects metadata with non-primitive values (nested object)", () => {
    const payload = {
      issues: [{ ...baseIssue, metadata: { nested: { x: 1 } } }],
      total: 1,
    };
    expect(ListIssuesResponseSchema.safeParse(payload).success).toBe(false);
  });
});

// The duplicate-issue branch in create-issue.tsx feeds ApiError.body
// (typed as `unknown`) through this schema. Any future server drift that
// loses the contract MUST fail the parse so the UI falls back to a normal
// error toast instead of rendering an empty / partial duplicate card.
describe("DuplicateIssueErrorBodySchema", () => {
  const valid = {
    code: "active_duplicate_issue",
    error: "An active issue with this title already exists: MUL-12 – Login bug",
    issue: {
      id: "11111111-1111-1111-1111-111111111111",
      identifier: "MUL-12",
      title: "Login bug",
    },
  };

  it("accepts a well-formed body", () => {
    expect(DuplicateIssueErrorBodySchema.safeParse(valid).success).toBe(true);
  });

  it("accepts unknown extra fields via .loose()", () => {
    const forwardCompat = {
      ...valid,
      hint: "Try a different title",
      issue: { ...valid.issue, workspace_id: "ws-1", status: "todo" },
    };
    expect(DuplicateIssueErrorBodySchema.safeParse(forwardCompat).success).toBe(true);
  });

  it("rejects a renamed code (so renames degrade to the generic toast)", () => {
    const renamed = { ...valid, code: "duplicate_issue" };
    expect(DuplicateIssueErrorBodySchema.safeParse(renamed).success).toBe(false);
  });

  it("rejects a missing issue object", () => {
    const { issue: _omit, ...without } = valid;
    expect(DuplicateIssueErrorBodySchema.safeParse(without).success).toBe(false);
  });

  it("rejects a non-string issue.id", () => {
    const broken = { ...valid, issue: { ...valid.issue, id: 42 } };
    expect(DuplicateIssueErrorBodySchema.safeParse(broken).success).toBe(false);
  });

  it("accepts a missing error field (it is optional)", () => {
    const { error: _omit, ...without } = valid;
    expect(DuplicateIssueErrorBodySchema.safeParse(without).success).toBe(true);
  });
});

// `user.timezone` (Viewing tz) was added in the timezone-architecture RFC.
// A desktop build older than the server — or a server predating the
// `user.timezone` migration — will return a `/api/me` body with no
// `timezone` key. The schema must not fail closed on that: the field
// defaults to `null`, which the frontend resolves to the browser-detected
// tz at render time.
describe("UserSchema timezone drift", () => {
  const base = {
    id: "11111111-1111-1111-1111-111111111111",
    name: "Ada",
    email: "ada@example.com",
  };

  it("defaults timezone to null when the field is absent", () => {
    const parsed = UserSchema.parse(base);
    expect(parsed.timezone).toBe(null);
  });

  it("preserves an explicit IANA timezone", () => {
    const parsed = UserSchema.parse({ ...base, timezone: "Asia/Tokyo" });
    expect(parsed.timezone).toBe("Asia/Tokyo");
  });

  it("accepts an explicit null timezone", () => {
    const parsed = UserSchema.parse({ ...base, timezone: null });
    expect(parsed.timezone).toBe(null);
  });

  // Wrong-type drift: a future server bug sending `timezone` as a number
  // must not throw into the UI. parseWithFallback degrades the whole user
  // object to the explicit fallback (EMPTY_USER) so /api/me callers keep a
  // valid shape instead of white-screening.
  it("falls back to EMPTY_USER when timezone is the wrong type", () => {
    const parsed = parseWithFallback(
      { ...base, timezone: 42 },
      UserSchema,
      EMPTY_USER,
      { endpoint: "GET /api/me" },
    );
    expect(parsed).toBe(EMPTY_USER);
  });
});


// The workspace dashboard and runtime-detail pages were re-pointed at the
// unified `task_usage_hourly` rollup. Every numeric field drives chart /
// KPI math, and string keys (date / agent_id / model) bucket the series.
// The contract these schemas must hold: a row missing a field degrades
// that field to a sane default rather than dropping the WHOLE array to
// the `[]` fallback — one drifted row must not blank the entire chart.
describe("dashboard + runtime usage schema drift", () => {
  it("coerces a missing numeric field to 0 instead of dropping the array", () => {
    const parsed = DashboardUsageDailyListSchema.parse([
      { date: "2026-05-19", model: "claude-opus-4-7", input_tokens: 100 },
    ]);
    expect(parsed).toHaveLength(1);
    expect(parsed[0]?.output_tokens).toBe(0);
    expect(parsed[0]?.cache_read_tokens).toBe(0);
    expect(parsed[0]?.cache_write_tokens).toBe(0);
  });

  it("coerces a missing date key to \"\" so the rest of the series survives", () => {
    const parsed = DashboardUsageDailyListSchema.parse([
      { model: "claude-opus-4-7", input_tokens: 5 },
    ]);
    expect(parsed).toHaveLength(1);
    expect(parsed[0]?.date).toBe("");
  });

  it("coerces a missing agent_id key to \"\" for the agent-runtime panel", () => {
    const parsed = DashboardAgentRunTimeListSchema.parse([
      { total_seconds: 42, task_count: 3, failed_count: 0 },
    ]);
    expect(parsed).toHaveLength(1);
    expect(parsed[0]?.agent_id).toBe("");
  });

  it("coerces a missing agent_id key to \"\" for the usage-by-agent panel", () => {
    const parsed = DashboardUsageByAgentListSchema.parse([
      { model: "claude-opus-4-7", input_tokens: 7 },
    ]);
    expect(parsed[0]?.agent_id).toBe("");
  });

  it("coerces missing fields on every runtime usage schema", () => {
    expect(RuntimeUsageListSchema.parse([{ date: "2026-05-19" }])[0]?.input_tokens).toBe(0);
    expect(RuntimeHourlyActivityListSchema.parse([{ hour: 9 }])[0]?.count).toBe(0);
    expect(RuntimeUsageByAgentListSchema.parse([{ model: "x" }])[0]?.agent_id).toBe("");
    expect(RuntimeUsageByHourListSchema.parse([{ hour: 9 }])[0]?.model).toBe("");
  });

  it("rejects a non-array body so parseWithFallback can return its fallback", () => {
    expect(DashboardUsageDailyListSchema.safeParse(null).success).toBe(false);
    expect(RuntimeUsageListSchema.safeParse({ rows: [] }).success).toBe(false);
  });

  it("keeps unknown server-side fields via .loose()", () => {
    const parsed = RuntimeUsageListSchema.parse([
      { date: "2026-05-19", region: "us-east" },
    ]);
    expect((parsed[0] as Record<string, unknown>).region).toBe("us-east");
  });
});

describe("FeatureIssuesResponseSchema", () => {
  it("defaults missing arrays to [] when fields are absent", () => {
    const result = parseWithFallback({}, FeatureIssuesResponseSchema, EMPTY_FEATURE_ISSUES_RESPONSE, { endpoint: "test" });
    expect(result.ready_now).toEqual([]);
    expect(result.blocked).toEqual([]);
    expect(result.pull_requests).toEqual([]);
  });

  it("falls back on null input", () => {
    const result = parseWithFallback(null, FeatureIssuesResponseSchema, EMPTY_FEATURE_ISSUES_RESPONSE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_FEATURE_ISSUES_RESPONSE);
  });

  it("parses a well-formed response with repo fields", () => {
    const input = {
      ready_now: [
        { id: "i1", identifier: "MUL-1", title: "Task", status: "todo", priority: "high", repo_id: "r1", repo_name: "backend" },
      ],
      blocked: [],
      pull_requests: [
        { number: 1, html_url: "https://github.com/owner/repo/pull/1", state: "open", title: "PR", repo_id: "r1" },
      ],
    };
    const result = parseWithFallback(input, FeatureIssuesResponseSchema, EMPTY_FEATURE_ISSUES_RESPONSE, { endpoint: "test" });
    expect(result.ready_now).toHaveLength(1);
    expect(result.ready_now[0]?.repo_name).toBe("backend");
    expect(result.pull_requests[0]?.repo_id).toBe("r1");
  });

  it("defaults blocked_by to [] when absent from a blocked issue", () => {
    const input = {
      ready_now: [],
      blocked: [{ id: "i2", identifier: "MUL-2", title: "Blocked", status: "backlog", priority: "medium" }],
      pull_requests: [],
    };
    const result = parseWithFallback(input, FeatureIssuesResponseSchema, EMPTY_FEATURE_ISSUES_RESPONSE, { endpoint: "test" });
    expect(result.blocked[0]?.blocked_by).toEqual([]);
  });
});

describe("FeatureSchema", () => {
  const baseFeature = {
    id: "f1",
    workspace_id: "ws-1",
    title: "Initiative",
    status: "running",
    priority: "none",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  };

  it("defaults Mode and tripwire fields when the server omits them", () => {
    const result = parseWithFallback(baseFeature, FeatureSchema, EMPTY_FEATURE, { endpoint: "test" });
    expect(result.mode).toBe("hitl");
    expect(result.budget_tokens).toBe(0);
    expect(result.budget_runs).toBe(0);
    expect(result.budget_seconds).toBe(0);
    expect(result.failure_tolerance).toBe(3);
  });

  it("preserves explicit Mode and budget values", () => {
    const input = { ...baseFeature, mode: "afk", budget_runs: 50, failure_tolerance: 2 };
    const result = parseWithFallback(input, FeatureSchema, EMPTY_FEATURE, { endpoint: "test" });
    expect(result.mode).toBe("afk");
    expect(result.budget_runs).toBe(50);
    expect(result.failure_tolerance).toBe(2);
  });

  it("falls back on null input", () => {
    const result = parseWithFallback(null, FeatureSchema, EMPTY_FEATURE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_FEATURE);
  });
});

describe("ListMilestonesResponseSchema", () => {
  const baseMilestone = {
    id: "m1",
    workspace_id: "ws-1",
    feature_id: "f1",
    title: "v1.0",
    position: 0,
    validation_status: "pending",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  };

  it("defaults milestones to [] when the field is absent", () => {
    const result = parseWithFallback({}, ListMilestonesResponseSchema, EMPTY_LIST_MILESTONES_RESPONSE, { endpoint: "test" });
    expect(result.milestones).toEqual([]);
    expect(result.total).toBe(0);
  });

  it("falls back on null input", () => {
    const result = parseWithFallback(null, ListMilestonesResponseSchema, EMPTY_LIST_MILESTONES_RESPONSE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_LIST_MILESTONES_RESPONSE);
  });

  it("parses a well-formed response and keeps an unknown validation_status", () => {
    const input = { milestones: [{ ...baseMilestone, validation_status: "in_review" }], total: 1 };
    const result = parseWithFallback(input, ListMilestonesResponseSchema, EMPTY_LIST_MILESTONES_RESPONSE, { endpoint: "test" });
    expect(result.milestones).toHaveLength(1);
    expect(result.milestones[0]?.validation_status).toBe("in_review");
  });

  it("falls back when a milestone is missing its required feature_id", () => {
    const { feature_id: _omit, ...broken } = baseMilestone;
    const input = { milestones: [broken], total: 1 };
    const result = parseWithFallback(input, ListMilestonesResponseSchema, EMPTY_LIST_MILESTONES_RESPONSE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_LIST_MILESTONES_RESPONSE);
  });

  // The single-Milestone create/update responses (control plane, issue 14).
  it("parses a single Milestone create/update response", () => {
    const result = parseWithFallback(baseMilestone, MilestoneSchema, EMPTY_MILESTONE, { endpoint: "test" });
    expect(result.id).toBe("m1");
    expect(result.title).toBe("v1.0");
  });

  it("falls back to EMPTY_MILESTONE on a malformed single Milestone", () => {
    const { id: _omit, ...broken } = baseMilestone;
    const result = parseWithFallback(broken, MilestoneSchema, EMPTY_MILESTONE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_MILESTONE);
  });
});

describe("HandoffSchema", () => {
  const baseHandoff = {
    id: "hid-1",
    workspace_id: "ws-1",
    issue_id: "issue-1",
    run_id: "run-1",
    done: ["step A", "step B"],
    left_undone: ["step C"],
    commands: [{ command: "go build ./...", exit_code: 0 }],
    discoveries: ["found bug in upstream"],
    created_at: "2026-01-01T00:00:00Z",
  };

  it("parses a well-formed handoff", () => {
    const result = HandoffSchema.safeParse(baseHandoff);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.done).toEqual(["step A", "step B"]);
      expect(result.data.commands).toHaveLength(1);
    }
  });

  it("defaults done/left_undone/discoveries to [] when absent", () => {
    const { done: _d, left_undone: _l, discoveries: _disc, ...partial } = baseHandoff;
    const result = HandoffSchema.safeParse(partial);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.done).toEqual([]);
      expect(result.data.left_undone).toEqual([]);
      expect(result.data.discoveries).toEqual([]);
    }
  });

  it("defaults commands to [] when absent", () => {
    const { commands: _c, ...partial } = baseHandoff;
    const result = HandoffSchema.safeParse(partial);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.commands).toEqual([]);
    }
  });

  it("falls back on null input via parseWithFallback", () => {
    const result = parseWithFallback(null, ListHandoffsResponseSchema, EMPTY_LIST_HANDOFFS_RESPONSE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_LIST_HANDOFFS_RESPONSE);
  });

  it("defaults handoffs to [] when field is absent", () => {
    const result = parseWithFallback({}, ListHandoffsResponseSchema, EMPTY_LIST_HANDOFFS_RESPONSE, { endpoint: "test" });
    expect(result.handoffs).toEqual([]);
  });

  it("falls back when required id field is missing", () => {
    const { id: _id, ...broken } = baseHandoff;
    const input = { handoffs: [broken] };
    const result = parseWithFallback(input, ListHandoffsResponseSchema, EMPTY_LIST_HANDOFFS_RESPONSE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_LIST_HANDOFFS_RESPONSE);
  });
});

describe("DecisionLogEntrySchema", () => {
  const baseEntry = {
    id: "dl-1",
    workspace_id: "ws-1",
    feature_id: "feat-1",
    run_id: "run-1",
    title: "Keep the Gate thin",
    decision: "SQL enforces, Go specifies",
    learning: "two layers stayed in sync",
    adr_refs: ["0004"],
    context_terms: ["Gate"],
    created_at: "2026-01-01T00:00:00Z",
  };

  it("parses a well-formed entry", () => {
    const result = DecisionLogEntrySchema.safeParse(baseEntry);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.adr_refs).toEqual(["0004"]);
      expect(result.data.context_terms).toEqual(["Gate"]);
    }
  });

  it("defaults learning/adr_refs/context_terms when absent", () => {
    const { learning: _le, adr_refs: _a, context_terms: _c, ...partial } = baseEntry;
    const result = DecisionLogEntrySchema.safeParse(partial);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.learning).toBe("");
      expect(result.data.adr_refs).toEqual([]);
      expect(result.data.context_terms).toEqual([]);
    }
  });

  it("falls back on null input via parseWithFallback", () => {
    const result = parseWithFallback(null, ListDecisionLogResponseSchema, EMPTY_LIST_DECISION_LOG_RESPONSE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_LIST_DECISION_LOG_RESPONSE);
  });

  it("defaults decisions to [] when field is absent", () => {
    const result = parseWithFallback({}, ListDecisionLogResponseSchema, EMPTY_LIST_DECISION_LOG_RESPONSE, { endpoint: "test" });
    expect(result.decisions).toEqual([]);
  });

  it("falls back when a required field is missing", () => {
    const { decision: _dec, ...broken } = baseEntry;
    const input = { decisions: [broken] };
    const result = parseWithFallback(input, ListDecisionLogResponseSchema, EMPTY_LIST_DECISION_LOG_RESPONSE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_LIST_DECISION_LOG_RESPONSE);
  });
});

describe("ListDodAssertionsResponseSchema", () => {
  const baseAssertion = {
    id: "a1",
    workspace_id: "ws-1",
    feature_id: "f1",
    milestone_id: "m1",
    text: "tests pass",
    position: 0,
    created_at: "2026-01-01T00:00:00Z",
    status: "passed",
    detail: "",
  };

  it("defaults assertions to [] when the field is absent", () => {
    const result = parseWithFallback({}, ListDodAssertionsResponseSchema, EMPTY_LIST_DOD_ASSERTIONS_RESPONSE, { endpoint: "test" });
    expect(result.assertions).toEqual([]);
  });

  it("falls back on null input", () => {
    const result = parseWithFallback(null, ListDodAssertionsResponseSchema, EMPTY_LIST_DOD_ASSERTIONS_RESPONSE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_LIST_DOD_ASSERTIONS_RESPONSE);
  });

  it("keeps an unknown status and defaults a missing detail", () => {
    const { detail: _omit, ...partial } = baseAssertion;
    const result = DodAssertionSchema.safeParse({ ...partial, status: "stale" });
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.status).toBe("stale");
      expect(result.data.detail).toBe("");
    }
  });

  it("falls back when an assertion is missing its required milestone_id", () => {
    const { milestone_id: _omit, ...broken } = baseAssertion;
    const input = { assertions: [broken] };
    const result = parseWithFallback(input, ListDodAssertionsResponseSchema, EMPTY_LIST_DOD_ASSERTIONS_RESPONSE, { endpoint: "test" });
    expect(result).toEqual(EMPTY_LIST_DOD_ASSERTIONS_RESPONSE);
  });

  // The single-assertion create response (control plane, issue 14).
  it("parses a single DoD assertion create response", () => {
    const result = parseWithFallback(baseAssertion, DodAssertionSchema, EMPTY_DOD_ASSERTION, { endpoint: "test" });
    expect(result.id).toBe("a1");
    expect(result.text).toBe("tests pass");
  });

  it("falls back to EMPTY_DOD_ASSERTION on a malformed single assertion", () => {
    const { milestone_id: _omit, ...broken } = baseAssertion;
    const result = parseWithFallback(broken, DodAssertionSchema, EMPTY_DOD_ASSERTION, { endpoint: "test" });
    expect(result).toEqual(EMPTY_DOD_ASSERTION);
  });
});
