-- Initiative Mode (HITL/AFK) and the per-Initiative budget + failure tolerance
-- (ADR-0005). Mode records the planning-time autonomy choice; the budget and
-- failure-tolerance fields feed the Tripwire/Budget safety net that pauses a
-- runaway Initiative — moving it to `blocked` and pinging the human — instead of
-- burning resources on an imperfect Definition of Done. A zero budget means
-- "no cap" for that dimension.

ALTER TABLE feature ADD COLUMN mode text NOT NULL DEFAULT 'hitl';
ALTER TABLE feature ADD CONSTRAINT feature_mode_check
    CHECK (mode = ANY (ARRAY['hitl', 'afk']));

ALTER TABLE feature ADD COLUMN budget_tokens bigint NOT NULL DEFAULT 0;
ALTER TABLE feature ADD COLUMN budget_runs integer NOT NULL DEFAULT 0;
ALTER TABLE feature ADD COLUMN budget_seconds bigint NOT NULL DEFAULT 0;
ALTER TABLE feature ADD COLUMN failure_tolerance integer NOT NULL DEFAULT 3;
