-- Remove member assignee type: null out any member-assigned issues, update constraint.

UPDATE public.issue
SET assignee_type = NULL, assignee_id = NULL
WHERE assignee_type = 'member';

ALTER TABLE public.issue DROP CONSTRAINT IF EXISTS issue_assignee_type_check;
ALTER TABLE public.issue ADD CONSTRAINT issue_assignee_type_check
    CHECK ((assignee_type = ANY (ARRAY['agent'::text])));
