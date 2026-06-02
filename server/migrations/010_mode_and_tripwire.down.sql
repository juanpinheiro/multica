-- Reverse 010: drop the Mode and Tripwire/Budget fields.

ALTER TABLE feature DROP CONSTRAINT IF EXISTS feature_mode_check;
ALTER TABLE feature DROP COLUMN IF EXISTS mode;
ALTER TABLE feature DROP COLUMN IF EXISTS budget_tokens;
ALTER TABLE feature DROP COLUMN IF EXISTS budget_runs;
ALTER TABLE feature DROP COLUMN IF EXISTS budget_seconds;
ALTER TABLE feature DROP COLUMN IF EXISTS failure_tolerance;
