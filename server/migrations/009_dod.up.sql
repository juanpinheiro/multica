-- Definition of Done (ADR-0007): initiative-level assertions, each tagged to a
-- Milestone. A validator Run checks a Milestone's accumulated work against its
-- assertions at the Milestone boundary; an Issue's per-Issue Acceptance Criteria
-- is the view of its Milestone's assertions.

CREATE TABLE dod_assertion (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    feature_id uuid NOT NULL REFERENCES feature(id) ON DELETE CASCADE,
    milestone_id uuid NOT NULL REFERENCES milestone(id) ON DELETE CASCADE,
    text text NOT NULL,
    position integer NOT NULL DEFAULT 0,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX dod_assertion_milestone_id_idx ON dod_assertion (milestone_id);
CREATE INDEX dod_assertion_feature_id_idx ON dod_assertion (feature_id);

-- A validator Run's verdict for one assertion. Re-validation Runs append new
-- rows; the latest verdict per assertion (newest created_at) is authoritative.
CREATE TABLE dod_assertion_result (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    assertion_id uuid NOT NULL REFERENCES dod_assertion(id) ON DELETE CASCADE,
    run_id uuid NOT NULL REFERENCES agent_task_queue(id) ON DELETE CASCADE,
    passed boolean NOT NULL,
    detail text NOT NULL DEFAULT '',
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX dod_assertion_result_assertion_id_idx ON dod_assertion_result (assertion_id);
CREATE INDEX dod_assertion_result_run_id_idx ON dod_assertion_result (run_id);
