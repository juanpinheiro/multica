-- Remove chat sessions, chat messages, quick-create support, and their
-- references from agent_task_queue and attachment.

-- Drop chat-related indexes before dropping tables/columns.
DROP INDEX IF EXISTS public.idx_chat_message_session;
DROP INDEX IF EXISTS public.idx_chat_session_creator;
DROP INDEX IF EXISTS public.idx_chat_session_workspace;
DROP INDEX IF EXISTS public.idx_agent_task_queue_chat_pending;
DROP INDEX IF EXISTS public.idx_attachment_chat_session;

-- Remove chat_session_id FK constraint from agent_task_queue before dropping
-- the table so the DROP TABLE succeeds.
ALTER TABLE public.agent_task_queue
    DROP CONSTRAINT IF EXISTS agent_task_queue_chat_session_id_fkey;

ALTER TABLE public.agent_task_queue
    DROP COLUMN IF EXISTS chat_session_id;

-- Remove chat-related columns from attachment.
ALTER TABLE public.attachment
    DROP COLUMN IF EXISTS chat_session_id,
    DROP COLUMN IF EXISTS chat_message_id;

-- Drop chat tables (chat_message references chat_session via FK).
DROP TABLE IF EXISTS public.chat_message;
DROP TABLE IF EXISTS public.chat_session;

-- Remove quick_create from issue.origin_type_check. The constraint must be
-- dropped and recreated because Postgres does not support ALTER CONSTRAINT.
ALTER TABLE public.issue
    DROP CONSTRAINT IF EXISTS issue_origin_type_check;

ALTER TABLE public.issue
    ADD CONSTRAINT issue_origin_type_check
    CHECK ((origin_type = ANY (ARRAY['autopilot'::text])));
