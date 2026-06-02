-- Reshape upstream tables toward the Initiative → Milestone → Issue → Run model
-- (ADR-0002). This migration is additive/semantic: the physical table names
-- (`feature`, `agent_task_queue`, `issue`) are unchanged, but `feature` gains
-- the Initiative status state machine, `agent_task_queue` gains a Run `role`,
-- and `issue` gains a `milestone_id` slot (FK added when the milestone table
-- lands in issue 07).

-- 1. Initiative status set on feature.
ALTER TABLE feature DROP CONSTRAINT feature_status_check;

UPDATE feature SET status = CASE status
    WHEN 'planned'     THEN 'draft'
    WHEN 'in_progress' THEN 'running'
    WHEN 'paused'      THEN 'blocked'
    WHEN 'completed'   THEN 'done'
    WHEN 'cancelled'   THEN 'cancelled'
    ELSE 'draft'
END;

ALTER TABLE feature ALTER COLUMN status SET DEFAULT 'draft';

ALTER TABLE feature ADD CONSTRAINT feature_status_check
    CHECK (status = ANY (ARRAY['draft', 'ready', 'running', 'in_review', 'done', 'blocked', 'cancelled']));

-- 2. Run role on agent_task_queue: every execution is a worker or a validator.
ALTER TABLE agent_task_queue ADD COLUMN role text NOT NULL DEFAULT 'worker';

ALTER TABLE agent_task_queue ADD CONSTRAINT agent_task_queue_role_check
    CHECK (role = ANY (ARRAY['worker', 'validator']));

-- 3. Issue milestone slot. Nullable and FK-less for now; issue 07 adds the
-- milestone table and the foreign key.
ALTER TABLE issue ADD COLUMN milestone_id uuid;
