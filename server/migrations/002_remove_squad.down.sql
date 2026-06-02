-- Restore squad feature (reverse of 002_remove_squad.up.sql).

CREATE TABLE IF NOT EXISTS public.squad (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    leader_id uuid NOT NULL,
    creator_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    archived_at timestamp with time zone,
    archived_by uuid,
    avatar_url text,
    instructions text DEFAULT ''::text NOT NULL
);

CREATE TABLE IF NOT EXISTS public.squad_member (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    squad_id uuid NOT NULL,
    member_type text NOT NULL,
    member_id uuid NOT NULL,
    role text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT squad_member_member_type_check CHECK ((member_type = ANY (ARRAY['agent'::text, 'member'::text])))
);

ALTER TABLE public.autopilot_run ADD COLUMN IF NOT EXISTS squad_id uuid;

ALTER TABLE public.issue DROP CONSTRAINT IF EXISTS issue_assignee_type_check;
ALTER TABLE public.issue ADD CONSTRAINT issue_assignee_type_check
    CHECK ((assignee_type = ANY (ARRAY['member'::text, 'agent'::text, 'squad'::text])));

ALTER TABLE public.autopilot DROP CONSTRAINT IF EXISTS autopilot_assignee_type_check;
ALTER TABLE public.autopilot ADD CONSTRAINT autopilot_assignee_type_check
    CHECK ((assignee_type = ANY (ARRAY['agent'::text, 'squad'::text])));
