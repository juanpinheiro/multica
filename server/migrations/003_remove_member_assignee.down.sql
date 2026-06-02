-- Restore member assignee type constraint.

ALTER TABLE public.issue DROP CONSTRAINT IF EXISTS issue_assignee_type_check;
ALTER TABLE public.issue ADD CONSTRAINT issue_assignee_type_check
    CHECK ((assignee_type = ANY (ARRAY['member'::text, 'agent'::text])));
