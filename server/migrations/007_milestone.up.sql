-- Milestone entity (ADR-0002): an ordered checkpoint within an Initiative
-- (feature). A Milestone's Issues are gated until the previous Milestone in the
-- Initiative has passed validation — that gate lives in ClaimAgentTask; this
-- migration only adds the table and wires issue.milestone_id to it.

CREATE TABLE milestone (
    id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    feature_id uuid NOT NULL REFERENCES feature(id) ON DELETE CASCADE,
    title text NOT NULL,
    -- Ordering within the Initiative. The gate compares positions to find the
    -- "previous" Milestone, so lower position = earlier.
    position integer NOT NULL DEFAULT 0,
    -- Validation lifecycle: a Milestone starts 'pending' and the validator Run
    -- (issue 09) flips it to 'passed' or 'failed'. 'passed' opens the gate for
    -- the next Milestone's Issues.
    validation_status text NOT NULL DEFAULT 'pending'
        CHECK (validation_status = ANY (ARRAY['pending', 'passed', 'failed'])),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX milestone_feature_id_idx ON milestone (feature_id);
CREATE INDEX milestone_workspace_id_idx ON milestone (workspace_id);

-- Wire the issue slot added FK-less in 006 to the new table. A deleted
-- Milestone leaves its Issues un-gated rather than cascading the delete.
ALTER TABLE issue
    ADD CONSTRAINT issue_milestone_id_fkey
    FOREIGN KEY (milestone_id) REFERENCES milestone(id) ON DELETE SET NULL;
