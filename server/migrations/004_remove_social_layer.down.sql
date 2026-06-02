-- Restore the social layer tables.

CREATE TABLE IF NOT EXISTS public.comment_reaction (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    comment_id uuid NOT NULL REFERENCES public.comment(id) ON DELETE CASCADE,
    workspace_id uuid NOT NULL REFERENCES public.workspace(id) ON DELETE CASCADE,
    actor_type text NOT NULL,
    actor_id uuid NOT NULL,
    emoji text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (comment_id, actor_type, actor_id, emoji)
);

CREATE TABLE IF NOT EXISTS public.issue_reaction (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    issue_id uuid NOT NULL REFERENCES public.issue(id) ON DELETE CASCADE,
    workspace_id uuid NOT NULL REFERENCES public.workspace(id) ON DELETE CASCADE,
    actor_type text NOT NULL,
    actor_id uuid NOT NULL,
    emoji text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (issue_id, actor_type, actor_id, emoji)
);

CREATE TABLE IF NOT EXISTS public.issue_subscriber (
    issue_id uuid NOT NULL REFERENCES public.issue(id) ON DELETE CASCADE,
    user_type text NOT NULL,
    user_id uuid NOT NULL,
    reason text NOT NULL CHECK (reason = ANY (ARRAY['creator'::text, 'assignee'::text, 'commenter'::text, 'mentioned'::text, 'manual'::text])),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (issue_id, user_type, user_id)
);

CREATE TABLE IF NOT EXISTS public.notification_preference (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL REFERENCES public.workspace(id) ON DELETE CASCADE,
    user_id uuid NOT NULL,
    preferences jsonb NOT NULL DEFAULT '{}',
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (workspace_id, user_id)
);
