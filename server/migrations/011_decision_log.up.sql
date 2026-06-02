-- Decision Log (CONTEXT.md "Decision Log"): the self-evolving layer of
-- architectural decisions kept current by agents. A retrospective Run at an
-- Initiative boundary revisits technical decisions, records what was learned,
-- and writes one row per decision. Entries link to the ADRs and CONTEXT terms
-- they touch so the durable architecture docs stay navigable.

CREATE TABLE decision_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    feature_id uuid NOT NULL REFERENCES feature(id) ON DELETE CASCADE,
    run_id uuid NOT NULL REFERENCES agent_task_queue(id) ON DELETE CASCADE,
    title text NOT NULL,
    decision text NOT NULL,
    learning text NOT NULL DEFAULT '',
    adr_refs text[] NOT NULL DEFAULT '{}',
    context_terms text[] NOT NULL DEFAULT '{}',
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX decision_log_feature_id_idx ON decision_log (feature_id);
CREATE INDEX decision_log_run_id_idx ON decision_log (run_id);

-- A retrospective is a third kind of Run alongside worker and validator: it
-- reviews the finished Initiative and writes the Decision Log instead of code.
ALTER TABLE agent_task_queue DROP CONSTRAINT agent_task_queue_role_check;

ALTER TABLE agent_task_queue ADD CONSTRAINT agent_task_queue_role_check
    CHECK (role = ANY (ARRAY['worker', 'validator', 'retrospective']));
