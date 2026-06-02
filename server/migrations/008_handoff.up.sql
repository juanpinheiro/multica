-- Handoff entity (ADR-0004): the structured output a worker Run writes when it
-- finishes an Issue. The Orchestrator reads handoffs on wake to resume fresh
-- context — what was done, what was left undone, which commands ran, and any
-- discoveries made. Handoffs are immutable once written (no updated_at).

CREATE TABLE handoff (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issue(id) ON DELETE CASCADE,
    run_id uuid NOT NULL REFERENCES agent_task_queue(id) ON DELETE CASCADE,
    -- What the Run accomplished.
    done text[] NOT NULL DEFAULT '{}',
    -- What was left incomplete for the next Run.
    left_undone text[] NOT NULL DEFAULT '{}',
    -- Commands executed and their exit codes (stored as JSONB array of
    -- {command: string, exit_code: int}).
    commands jsonb NOT NULL DEFAULT '[]',
    -- Unexpected findings surfaced during the Run.
    discoveries text[] NOT NULL DEFAULT '{}',
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX handoff_issue_id_idx ON handoff (issue_id);
CREATE INDEX handoff_run_id_idx ON handoff (run_id);
