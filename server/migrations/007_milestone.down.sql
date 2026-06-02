-- Reverse 007: detach issues from milestones and drop the table.

ALTER TABLE issue DROP CONSTRAINT issue_milestone_id_fkey;

DROP TABLE milestone;
