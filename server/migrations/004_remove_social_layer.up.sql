-- Remove the human-collaboration social layer: reactions, subscribers, notification preferences.

DROP TABLE IF EXISTS public.comment_reaction CASCADE;
DROP TABLE IF EXISTS public.issue_reaction CASCADE;
DROP TABLE IF EXISTS public.issue_subscriber CASCADE;
DROP TABLE IF EXISTS public.notification_preference CASCADE;
