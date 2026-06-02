-- Reverse 006: restore the upstream feature status set and drop the Run role
-- and issue milestone slot.

ALTER TABLE issue DROP COLUMN milestone_id;

ALTER TABLE agent_task_queue DROP CONSTRAINT agent_task_queue_role_check;
ALTER TABLE agent_task_queue DROP COLUMN role;

ALTER TABLE feature DROP CONSTRAINT feature_status_check;

UPDATE feature SET status = CASE status
    WHEN 'draft'     THEN 'planned'
    WHEN 'ready'     THEN 'planned'
    WHEN 'running'   THEN 'in_progress'
    WHEN 'in_review' THEN 'in_progress'
    WHEN 'done'      THEN 'completed'
    WHEN 'blocked'   THEN 'paused'
    WHEN 'cancelled' THEN 'cancelled'
    ELSE 'planned'
END;

ALTER TABLE feature ALTER COLUMN status SET DEFAULT 'planned';

ALTER TABLE feature ADD CONSTRAINT feature_status_check
    CHECK (status = ANY (ARRAY['planned', 'in_progress', 'paused', 'completed', 'cancelled']));
