-- Restore chat sessions, chat messages, quick-create support, and their
-- references in agent_task_queue and attachment.

ALTER TABLE public.issue
    DROP CONSTRAINT IF EXISTS issue_origin_type_check;

ALTER TABLE public.issue
    ADD CONSTRAINT issue_origin_type_check
    CHECK ((origin_type = ANY (ARRAY['autopilot'::text, 'quick_create'::text])));

CREATE TABLE public.chat_session (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    agent_id uuid NOT NULL,
    creator_id uuid NOT NULL,
    title text NOT NULL DEFAULT '',
    session_id text,
    work_dir text,
    status text NOT NULL DEFAULT 'active',
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    unread_since timestamp with time zone,
    runtime_id uuid,
    CONSTRAINT chat_session_pkey PRIMARY KEY (id),
    CONSTRAINT chat_session_status_check CHECK ((status = ANY (ARRAY['active'::text, 'archived'::text])))
);

CREATE TABLE public.chat_message (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    chat_session_id uuid NOT NULL,
    role text NOT NULL,
    content text NOT NULL DEFAULT '',
    task_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    failure_reason text,
    elapsed_ms integer,
    CONSTRAINT chat_message_pkey PRIMARY KEY (id),
    CONSTRAINT chat_message_chat_session_id_fkey FOREIGN KEY (chat_session_id)
        REFERENCES public.chat_session(id) ON DELETE CASCADE
);

ALTER TABLE public.attachment
    ADD COLUMN chat_session_id uuid,
    ADD COLUMN chat_message_id uuid;

ALTER TABLE public.agent_task_queue
    ADD COLUMN chat_session_id uuid;

ALTER TABLE public.agent_task_queue
    ADD CONSTRAINT agent_task_queue_chat_session_id_fkey
        FOREIGN KEY (chat_session_id) REFERENCES public.chat_session(id) ON DELETE SET NULL;

CREATE INDEX idx_chat_message_session ON public.chat_message USING btree (chat_session_id, created_at);
CREATE INDEX idx_chat_session_creator ON public.chat_session USING btree (creator_id, workspace_id);
CREATE INDEX idx_chat_session_workspace ON public.chat_session USING btree (workspace_id);
CREATE INDEX idx_agent_task_queue_chat_pending ON public.agent_task_queue
    USING btree (chat_session_id, created_at DESC)
    WHERE ((chat_session_id IS NOT NULL) AND (status = ANY (ARRAY['queued'::text, 'dispatched'::text, 'running'::text])));
CREATE INDEX idx_attachment_chat_session ON public.attachment
    USING btree (chat_session_id)
    WHERE (chat_session_id IS NOT NULL);
