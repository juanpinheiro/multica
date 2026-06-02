-- Remove squad feature: drop squad/squad_member tables and update assignee_type constraints.

DROP TABLE IF EXISTS public.squad_member CASCADE;
DROP TABLE IF EXISTS public.squad CASCADE;

ALTER TABLE public.autopilot_run DROP COLUMN IF EXISTS squad_id;

ALTER TABLE public.issue DROP CONSTRAINT IF EXISTS issue_assignee_type_check;
ALTER TABLE public.issue ADD CONSTRAINT issue_assignee_type_check
    CHECK ((assignee_type = ANY (ARRAY['member'::text, 'agent'::text])));

ALTER TABLE public.autopilot DROP CONSTRAINT IF EXISTS autopilot_assignee_type_check;
ALTER TABLE public.autopilot ADD CONSTRAINT autopilot_assignee_type_check
    CHECK ((assignee_type = ANY (ARRAY['agent'::text])));
