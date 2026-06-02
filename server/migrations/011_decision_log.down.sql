UPDATE agent_task_queue SET role = 'worker' WHERE role = 'retrospective';

ALTER TABLE agent_task_queue DROP CONSTRAINT agent_task_queue_role_check;

ALTER TABLE agent_task_queue ADD CONSTRAINT agent_task_queue_role_check
    CHECK (role = ANY (ARRAY['worker', 'validator']));

DROP TABLE IF EXISTS decision_log;
