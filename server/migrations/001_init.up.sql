-- Skip function-body validation so SQL-language functions can reference
-- tables that are created later in this script.
SET check_function_bodies = false;

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

-- pg_cron is used by the task_usage hourly rollup pipeline. Wrap in DO/EXCEPTION
-- so dev/CI Postgres images without shared_preload_libraries=pg_cron skip
-- gracefully. Scheduling the rollup tick remains a deploy-time operator action.
DO $do$
BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_cron;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_cron extension not available; skipping.';
END
$do$;


--
-- Name: enqueue_task_usage_hourly_dirty_for_atq(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.enqueue_task_usage_hourly_dirty_for_atq() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF OLD.runtime_id IS DISTINCT FROM NEW.runtime_id
           OR OLD.issue_id IS DISTINCT FROM NEW.issue_id THEN
            -- OLD side. NULL runtime_id rows are not aggregated (no
            -- runtime → no bucket); skip those.
            IF OLD.runtime_id IS NOT NULL THEN
                INSERT INTO task_usage_hourly_dirty (
                    bucket_hour, workspace_id, runtime_id, agent_id,
                    feature_id, provider, model
                )
                SELECT DISTINCT
                    task_usage_hour_bucket(tu.created_at),
                    a.workspace_id,
                    OLD.runtime_id,
                    OLD.agent_id,
                    i_old.feature_id,
                    tu.provider,
                    tu.model
                  FROM task_usage tu
                  JOIN agent a ON a.id = OLD.agent_id
                  LEFT JOIN issue i_old ON i_old.id = OLD.issue_id
                 WHERE tu.task_id = OLD.id
                ON CONFLICT ON CONSTRAINT uq_task_usage_hourly_dirty_key DO UPDATE
                    SET enqueued_at = GREATEST(task_usage_hourly_dirty.enqueued_at, EXCLUDED.enqueued_at);
            END IF;

            IF NEW.runtime_id IS NOT NULL THEN
                INSERT INTO task_usage_hourly_dirty (
                    bucket_hour, workspace_id, runtime_id, agent_id,
                    feature_id, provider, model
                )
                SELECT DISTINCT
                    task_usage_hour_bucket(tu.created_at),
                    a.workspace_id,
                    NEW.runtime_id,
                    NEW.agent_id,
                    i_new.feature_id,
                    tu.provider,
                    tu.model
                  FROM task_usage tu
                  JOIN agent a ON a.id = NEW.agent_id
                  LEFT JOIN issue i_new ON i_new.id = NEW.issue_id
                 WHERE tu.task_id = NEW.id
                ON CONFLICT ON CONSTRAINT uq_task_usage_hourly_dirty_key DO UPDATE
                    SET enqueued_at = GREATEST(task_usage_hourly_dirty.enqueued_at, EXCLUDED.enqueued_at);
            END IF;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.runtime_id IS NOT NULL THEN
            INSERT INTO task_usage_hourly_dirty (
                bucket_hour, workspace_id, runtime_id, agent_id,
                feature_id, provider, model
            )
            SELECT DISTINCT
                task_usage_hour_bucket(tu.created_at),
                a.workspace_id,
                OLD.runtime_id,
                OLD.agent_id,
                i.feature_id,
                tu.provider,
                tu.model
              FROM task_usage tu
              JOIN agent a ON a.id = OLD.agent_id
              LEFT JOIN issue i ON i.id = OLD.issue_id
             WHERE tu.task_id = OLD.id
            ON CONFLICT ON CONSTRAINT uq_task_usage_hourly_dirty_key DO UPDATE
                SET enqueued_at = GREATEST(task_usage_hourly_dirty.enqueued_at, EXCLUDED.enqueued_at);
        END IF;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$;


--
-- Name: enqueue_task_usage_hourly_dirty_for_issue_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.enqueue_task_usage_hourly_dirty_for_issue_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO task_usage_hourly_dirty (
        bucket_hour, workspace_id, runtime_id, agent_id,
        feature_id, provider, model
    )
    SELECT DISTINCT
        task_usage_hour_bucket(tu.created_at),
        OLD.workspace_id,
        atq.runtime_id,
        atq.agent_id,
        OLD.feature_id,
        tu.provider,
        tu.model
      FROM agent_task_queue atq
      JOIN task_usage tu ON tu.task_id = atq.id
     WHERE atq.issue_id = OLD.id
       AND atq.runtime_id IS NOT NULL
    ON CONFLICT ON CONSTRAINT uq_task_usage_hourly_dirty_key DO UPDATE
        SET enqueued_at = GREATEST(task_usage_hourly_dirty.enqueued_at, EXCLUDED.enqueued_at);
    RETURN OLD;
END;
$$;


--
-- Name: enqueue_task_usage_hourly_dirty_for_issue_feature(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.enqueue_task_usage_hourly_dirty_for_issue_feature() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF OLD.feature_id IS DISTINCT FROM NEW.feature_id THEN
        -- OLD feature buckets.
        INSERT INTO task_usage_hourly_dirty (
            bucket_hour, workspace_id, runtime_id, agent_id,
            feature_id, provider, model
        )
        SELECT DISTINCT
            task_usage_hour_bucket(tu.created_at),
            NEW.workspace_id,
            atq.runtime_id,
            atq.agent_id,
            OLD.feature_id,
            tu.provider,
            tu.model
          FROM agent_task_queue atq
          JOIN task_usage tu ON tu.task_id = atq.id
         WHERE atq.issue_id = NEW.id
           AND atq.runtime_id IS NOT NULL
        ON CONFLICT ON CONSTRAINT uq_task_usage_hourly_dirty_key DO UPDATE
            SET enqueued_at = GREATEST(task_usage_hourly_dirty.enqueued_at, EXCLUDED.enqueued_at);

        -- NEW feature buckets.
        INSERT INTO task_usage_hourly_dirty (
            bucket_hour, workspace_id, runtime_id, agent_id,
            feature_id, provider, model
        )
        SELECT DISTINCT
            task_usage_hour_bucket(tu.created_at),
            NEW.workspace_id,
            atq.runtime_id,
            atq.agent_id,
            NEW.feature_id,
            tu.provider,
            tu.model
          FROM agent_task_queue atq
          JOIN task_usage tu ON tu.task_id = atq.id
         WHERE atq.issue_id = NEW.id
           AND atq.runtime_id IS NOT NULL
        ON CONFLICT ON CONSTRAINT uq_task_usage_hourly_dirty_key DO UPDATE
            SET enqueued_at = GREATEST(task_usage_hourly_dirty.enqueued_at, EXCLUDED.enqueued_at);
    END IF;
    RETURN NEW;
END;
$$;


--
-- Name: enqueue_task_usage_hourly_dirty_for_tu(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.enqueue_task_usage_hourly_dirty_for_tu() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO task_usage_hourly_dirty (
        bucket_hour, workspace_id, runtime_id, agent_id,
        feature_id, provider, model
    )
    SELECT
        task_usage_hour_bucket(OLD.created_at),
        a.workspace_id,
        atq.runtime_id,
        atq.agent_id,
        i.feature_id,
        OLD.provider,
        OLD.model
      FROM agent_task_queue atq
      JOIN agent a ON a.id = atq.agent_id
      LEFT JOIN issue i ON i.id = atq.issue_id
     WHERE atq.id = OLD.task_id
       AND atq.runtime_id IS NOT NULL
    ON CONFLICT ON CONSTRAINT uq_task_usage_hourly_dirty_key DO UPDATE
        SET enqueued_at = GREATEST(task_usage_hourly_dirty.enqueued_at, EXCLUDED.enqueued_at);
    RETURN OLD;
END;
$$;


--
-- Name: prune_task_usage_hourly_dirty(interval); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.prune_task_usage_hourly_dirty(p_retention interval DEFAULT '7 days'::interval) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_rows BIGINT;
BEGIN
    DELETE FROM task_usage_hourly_dirty
     WHERE enqueued_at < now() - p_retention;
    GET DIAGNOSTICS v_rows = ROW_COUNT;
    RETURN v_rows;
END;
$$;


--
-- Name: rollup_task_usage_hourly(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.rollup_task_usage_hourly() RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_lock_ok BOOLEAN;
    v_from    TIMESTAMPTZ;
    v_to      TIMESTAMPTZ;
    v_rows    BIGINT := 0;
BEGIN
    SELECT pg_try_advisory_lock(4246) INTO v_lock_ok;
    IF NOT v_lock_ok THEN
        RETURN 0;
    END IF;

    BEGIN
        UPDATE task_usage_hourly_rollup_state
           SET last_run_started_at = now(),
               last_error          = NULL
         WHERE id = 1
        RETURNING watermark_at INTO v_from;

        -- Cap each tick at a one-day window. In steady state v_from is
        -- recent, so LEAST picks `now() - 5 min` and nothing changes. But
        -- if the worker was paused (incident, migration freeze) the
        -- watermark can fall far behind; without the cap a single catch-up
        -- tick would recompute a multi-week window in one statement while
        -- holding lock 4246, blocking every other tick. Capped, catch-up
        -- advances in bounded one-day steps over successive ticks.
        v_to := LEAST(now() - INTERVAL '5 minutes', v_from + INTERVAL '1 day');

        IF v_from < v_to THEN
            v_rows := rollup_task_usage_hourly_window(v_from, v_to);

            UPDATE task_usage_hourly_rollup_state
               SET watermark_at         = v_to,
                   last_run_finished_at = now(),
                   last_run_rows        = v_rows
             WHERE id = 1;
        ELSE
            UPDATE task_usage_hourly_rollup_state
               SET last_run_finished_at = now(),
                   last_run_rows        = 0
             WHERE id = 1;
        END IF;

        PERFORM pg_advisory_unlock(4246);
    EXCEPTION WHEN OTHERS THEN
        UPDATE task_usage_hourly_rollup_state
           SET last_error           = SQLERRM,
               last_run_finished_at = now()
         WHERE id = 1;
        PERFORM pg_advisory_unlock(4246);
        RAISE;
    END;

    -- TTL prune. Runs AFTER the advisory lock is released: on a large
    -- stale backlog the prune can be slow, and holding lock 4246 through
    -- it would serialise every concurrent cron tick. It is a plain
    -- bounded DELETE — idempotent and safe to run unlocked.
    PERFORM prune_task_usage_hourly_dirty();
    RETURN v_rows;
END;
$$;


--
-- Name: rollup_task_usage_hourly_window(timestamp with time zone, timestamp with time zone); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.rollup_task_usage_hourly_window(p_from timestamp with time zone, p_to timestamp with time zone) RETURNS bigint
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_rows BIGINT;
BEGIN
    IF p_from >= p_to THEN
        RETURN 0;
    END IF;

    WITH
    dirty_from_updates AS (
        SELECT DISTINCT
            task_usage_hour_bucket(tu.created_at) AS bucket_hour,
            a.workspace_id                        AS workspace_id,
            atq.runtime_id                        AS runtime_id,
            atq.agent_id                          AS agent_id,
            i.feature_id                          AS feature_id,
            tu.provider                           AS provider,
            tu.model                              AS model
          FROM task_usage tu
          JOIN agent_task_queue atq ON atq.id      = tu.task_id
          JOIN agent            a   ON a.id        = atq.agent_id
          LEFT JOIN issue       i   ON i.id        = atq.issue_id
         WHERE atq.runtime_id IS NOT NULL
           AND (
                (tu.updated_at >= p_from AND tu.updated_at < p_to)
                -- Legacy updated_at-NULL rows; partial index from 078.
                OR (tu.updated_at IS NULL
                    AND tu.created_at >= p_from
                    AND tu.created_at <  p_to)
           )
    ),
    dirty_from_queue AS (
        SELECT bucket_hour, workspace_id, runtime_id, agent_id,
               feature_id, provider, model
          FROM task_usage_hourly_dirty
         WHERE enqueued_at < p_to
    ),
    dirty_keys AS (
        SELECT * FROM dirty_from_updates
        UNION
        SELECT * FROM dirty_from_queue
    ),
    recomputed AS (
        SELECT
            dk.bucket_hour,
            dk.workspace_id,
            dk.runtime_id,
            dk.agent_id,
            dk.feature_id,
            dk.provider,
            dk.model,
            SUM(tu.input_tokens)::bigint       AS input_tokens,
            SUM(tu.output_tokens)::bigint      AS output_tokens,
            SUM(tu.cache_read_tokens)::bigint  AS cache_read_tokens,
            SUM(tu.cache_write_tokens)::bigint AS cache_write_tokens,
            COUNT(DISTINCT tu.task_id)::bigint AS task_count,
            COUNT(*)::bigint                   AS event_count
          FROM dirty_keys dk
          JOIN agent_task_queue atq ON atq.runtime_id  = dk.runtime_id
                                    AND atq.agent_id    = dk.agent_id
          JOIN agent            a   ON a.id            = atq.agent_id
                                    AND a.workspace_id = dk.workspace_id
          LEFT JOIN issue       i   ON i.id            = atq.issue_id
          JOIN task_usage       tu  ON tu.task_id      = atq.id
                                    AND tu.provider    = dk.provider
                                    AND tu.model       = dk.model
                                    AND task_usage_hour_bucket(tu.created_at) = dk.bucket_hour
         WHERE (i.feature_id IS NOT DISTINCT FROM dk.feature_id)
         GROUP BY 1, 2, 3, 4, 5, 6, 7
    ),
    upserted AS (
        INSERT INTO task_usage_hourly AS d (
            bucket_hour, workspace_id, runtime_id, agent_id,
            feature_id, provider, model,
            input_tokens, output_tokens, cache_read_tokens, cache_write_tokens,
            task_count, event_count
        )
        SELECT
            bucket_hour, workspace_id, runtime_id, agent_id,
            feature_id, provider, model,
            input_tokens, output_tokens, cache_read_tokens, cache_write_tokens,
            task_count, event_count
          FROM recomputed
        ON CONFLICT ON CONSTRAINT uq_task_usage_hourly_key DO UPDATE
            SET input_tokens       = EXCLUDED.input_tokens,
                output_tokens      = EXCLUDED.output_tokens,
                cache_read_tokens  = EXCLUDED.cache_read_tokens,
                cache_write_tokens = EXCLUDED.cache_write_tokens,
                task_count         = EXCLUDED.task_count,
                event_count        = EXCLUDED.event_count,
                updated_at         = now()
        RETURNING 1
    ),
    deleted_empty AS (
        DELETE FROM task_usage_hourly d
         USING dirty_keys dk
         WHERE d.bucket_hour  = dk.bucket_hour
           AND d.workspace_id = dk.workspace_id
           AND d.runtime_id   = dk.runtime_id
           AND d.agent_id     = dk.agent_id
           AND d.feature_id IS NOT DISTINCT FROM dk.feature_id
           AND d.provider     = dk.provider
           AND d.model        = dk.model
           AND NOT EXISTS (
               SELECT 1 FROM recomputed r
                WHERE r.bucket_hour  = dk.bucket_hour
                  AND r.workspace_id = dk.workspace_id
                  AND r.runtime_id   = dk.runtime_id
                  AND r.agent_id     = dk.agent_id
                  AND r.feature_id IS NOT DISTINCT FROM dk.feature_id
                  AND r.provider     = dk.provider
                  AND r.model        = dk.model
           )
        RETURNING 1
    )
    SELECT (SELECT COUNT(*) FROM upserted) + (SELECT COUNT(*) FROM deleted_empty)
      INTO v_rows;

    DELETE FROM task_usage_hourly_dirty WHERE enqueued_at < p_to;

    RETURN v_rows;
END;
$$;


--
-- Name: task_usage_hour_bucket(timestamp with time zone); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.task_usage_hour_bucket(ts timestamp with time zone) RETURNS timestamp with time zone
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT (date_trunc('hour', ts AT TIME ZONE 'UTC')) AT TIME ZONE 'UTC';
$$;


--
-- Name: task_usage_hourly_rollup_lag_seconds(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.task_usage_hourly_rollup_lag_seconds() RETURNS double precision
    LANGUAGE sql STABLE
    AS $$
    SELECT EXTRACT(EPOCH FROM (now() - last_run_finished_at))
      FROM task_usage_hourly_rollup_state
     WHERE id = 1;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: activity_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.activity_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    issue_id uuid,
    actor_type text,
    actor_id uuid,
    action text NOT NULL,
    details jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT activity_log_actor_type_check CHECK ((actor_type = ANY (ARRAY['member'::text, 'agent'::text, 'system'::text])))
);


--
-- Name: agent; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    name text NOT NULL,
    avatar_url text,
    runtime_mode text NOT NULL,
    runtime_config jsonb DEFAULT '{}'::jsonb NOT NULL,
    visibility text DEFAULT 'private'::text NOT NULL,
    status text DEFAULT 'offline'::text NOT NULL,
    max_concurrent_tasks integer DEFAULT 6 NOT NULL,
    owner_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    runtime_id uuid NOT NULL,
    instructions text DEFAULT ''::text NOT NULL,
    archived_at timestamp with time zone,
    archived_by uuid,
    custom_env jsonb DEFAULT '{}'::jsonb NOT NULL,
    custom_args jsonb DEFAULT '[]'::jsonb NOT NULL,
    mcp_config jsonb,
    model text,
    thinking_level text,
    CONSTRAINT agent_description_length CHECK ((char_length(description) <= 255)),
    CONSTRAINT agent_runtime_mode_check CHECK ((runtime_mode = ANY (ARRAY['local'::text, 'cloud'::text]))),
    CONSTRAINT agent_status_check CHECK ((status = ANY (ARRAY['idle'::text, 'working'::text, 'blocked'::text, 'error'::text, 'offline'::text]))),
    CONSTRAINT agent_visibility_check CHECK ((visibility = ANY (ARRAY['workspace'::text, 'private'::text])))
);


--
-- Name: agent_runtime; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_runtime (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    daemon_id text,
    name text NOT NULL,
    runtime_mode text NOT NULL,
    provider text NOT NULL,
    status text DEFAULT 'offline'::text NOT NULL,
    device_info text DEFAULT ''::text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    last_seen_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    owner_id uuid,
    legacy_daemon_id text,
    visibility text DEFAULT 'private'::text NOT NULL,
    CONSTRAINT agent_runtime_runtime_mode_check CHECK ((runtime_mode = ANY (ARRAY['local'::text, 'cloud'::text]))),
    CONSTRAINT agent_runtime_status_check CHECK ((status = ANY (ARRAY['online'::text, 'offline'::text]))),
    CONSTRAINT agent_runtime_visibility_check CHECK ((visibility = ANY (ARRAY['private'::text, 'public'::text])))
);


--
-- Name: agent_skill; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_skill (
    agent_id uuid NOT NULL,
    skill_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: agent_task_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_task_queue (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agent_id uuid NOT NULL,
    issue_id uuid,
    status text DEFAULT 'queued'::text NOT NULL,
    priority integer DEFAULT 0 NOT NULL,
    dispatched_at timestamp with time zone,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    result jsonb,
    error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    context jsonb,
    runtime_id uuid NOT NULL,
    session_id text,
    work_dir text,
    trigger_comment_id uuid,
    chat_session_id uuid,
    autopilot_run_id uuid,
    attempt integer DEFAULT 1 NOT NULL,
    max_attempts integer DEFAULT 2 NOT NULL,
    parent_task_id uuid,
    failure_reason text,
    trigger_summary text,
    force_fresh_session boolean DEFAULT false NOT NULL,
    is_leader_task boolean DEFAULT false NOT NULL,
    CONSTRAINT agent_task_queue_status_check CHECK ((status = ANY (ARRAY['queued'::text, 'dispatched'::text, 'running'::text, 'completed'::text, 'failed'::text, 'cancelled'::text])))
);


--
-- Name: attachment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.attachment (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    issue_id uuid,
    comment_id uuid,
    uploader_type text NOT NULL,
    uploader_id uuid NOT NULL,
    filename text NOT NULL,
    url text NOT NULL,
    content_type text NOT NULL,
    size_bytes bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    chat_session_id uuid,
    chat_message_id uuid,
    CONSTRAINT attachment_uploader_type_check CHECK ((uploader_type = ANY (ARRAY['member'::text, 'agent'::text])))
);


--
-- Name: autopilot; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.autopilot (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    title text NOT NULL,
    description text,
    assignee_id uuid NOT NULL,
    status text DEFAULT 'active'::text NOT NULL,
    execution_mode text DEFAULT 'create_issue'::text NOT NULL,
    issue_title_template text,
    created_by_type text NOT NULL,
    created_by_id uuid NOT NULL,
    last_run_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    assignee_type text DEFAULT 'agent'::text NOT NULL,
    feature_id uuid,
    CONSTRAINT autopilot_assignee_type_check CHECK ((assignee_type = ANY (ARRAY['agent'::text, 'squad'::text]))),
    CONSTRAINT autopilot_created_by_type_check CHECK ((created_by_type = ANY (ARRAY['member'::text, 'agent'::text]))),
    CONSTRAINT autopilot_execution_mode_check CHECK ((execution_mode = ANY (ARRAY['create_issue'::text, 'run_only'::text]))),
    CONSTRAINT autopilot_status_check CHECK ((status = ANY (ARRAY['active'::text, 'paused'::text, 'archived'::text])))
);


--
-- Name: autopilot_run; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.autopilot_run (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    autopilot_id uuid NOT NULL,
    trigger_id uuid,
    source text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    issue_id uuid,
    task_id uuid,
    triggered_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    failure_reason text,
    trigger_payload jsonb,
    result jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    squad_id uuid,
    CONSTRAINT autopilot_run_source_check CHECK ((source = ANY (ARRAY['schedule'::text, 'manual'::text, 'webhook'::text, 'api'::text]))),
    CONSTRAINT autopilot_run_status_check CHECK ((status = ANY (ARRAY['issue_created'::text, 'running'::text, 'completed'::text, 'failed'::text, 'skipped'::text])))
);


--
-- Name: autopilot_trigger; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.autopilot_trigger (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    autopilot_id uuid NOT NULL,
    kind text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    cron_expression text,
    timezone text DEFAULT 'UTC'::text,
    next_run_at timestamp with time zone,
    webhook_token text,
    label text,
    last_fired_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    provider text DEFAULT 'generic'::text NOT NULL,
    signing_secret text,
    CONSTRAINT autopilot_trigger_kind_check CHECK ((kind = ANY (ARRAY['schedule'::text, 'webhook'::text, 'api'::text]))),
    CONSTRAINT autopilot_trigger_provider_check CHECK ((provider = ANY (ARRAY['generic'::text, 'github'::text])))
);


--
-- Name: chat_message; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_message (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    chat_session_id uuid NOT NULL,
    role text NOT NULL,
    content text NOT NULL,
    task_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    failure_reason text,
    elapsed_ms bigint,
    CONSTRAINT chat_message_role_check CHECK ((role = ANY (ARRAY['user'::text, 'assistant'::text])))
);


--
-- Name: chat_session; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_session (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    agent_id uuid NOT NULL,
    creator_id uuid NOT NULL,
    title text DEFAULT ''::text NOT NULL,
    session_id text,
    work_dir text,
    status text DEFAULT 'active'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    unread_since timestamp with time zone,
    runtime_id uuid,
    CONSTRAINT chat_session_status_check CHECK ((status = ANY (ARRAY['active'::text, 'archived'::text])))
);


--
-- Name: comment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.comment (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    issue_id uuid NOT NULL,
    author_type text NOT NULL,
    author_id uuid NOT NULL,
    content text NOT NULL,
    type text DEFAULT 'comment'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    parent_id uuid,
    workspace_id uuid NOT NULL,
    resolved_at timestamp with time zone,
    resolved_by_type text,
    resolved_by_id uuid,
    CONSTRAINT comment_author_type_check CHECK ((author_type = ANY (ARRAY['member'::text, 'agent'::text, 'system'::text]))),
    CONSTRAINT comment_resolved_consistency CHECK ((((resolved_at IS NULL) AND (resolved_by_type IS NULL) AND (resolved_by_id IS NULL)) OR ((resolved_at IS NOT NULL) AND (resolved_by_type IS NOT NULL) AND (resolved_by_id IS NOT NULL)))),
    CONSTRAINT comment_type_check CHECK ((type = ANY (ARRAY['comment'::text, 'status_change'::text, 'progress_update'::text, 'system'::text])))
);


--
-- Name: comment_reaction; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.comment_reaction (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    comment_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    actor_type text NOT NULL,
    actor_id uuid NOT NULL,
    emoji text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT comment_reaction_actor_type_check CHECK ((actor_type = ANY (ARRAY['member'::text, 'agent'::text])))
);


--
-- Name: daemon_connection; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.daemon_connection (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agent_id uuid NOT NULL,
    daemon_id text NOT NULL,
    status text DEFAULT 'disconnected'::text NOT NULL,
    last_heartbeat_at timestamp with time zone,
    runtime_info jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT daemon_connection_status_check CHECK ((status = ANY (ARRAY['connected'::text, 'disconnected'::text])))
);


--
-- Name: daemon_token; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.daemon_token (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    token_hash text NOT NULL,
    workspace_id uuid NOT NULL,
    daemon_id text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: github_installation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.github_installation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    installation_id bigint NOT NULL,
    account_login text NOT NULL,
    account_type text DEFAULT 'User'::text NOT NULL,
    account_avatar_url text,
    connected_by_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT github_installation_account_type_check CHECK ((account_type = ANY (ARRAY['User'::text, 'Organization'::text])))
);


--
-- Name: github_pull_request; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.github_pull_request (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    installation_id bigint NOT NULL,
    repo_owner text NOT NULL,
    repo_name text NOT NULL,
    pr_number integer NOT NULL,
    title text NOT NULL,
    state text NOT NULL,
    html_url text NOT NULL,
    branch text,
    author_login text,
    author_avatar_url text,
    merged_at timestamp with time zone,
    closed_at timestamp with time zone,
    pr_created_at timestamp with time zone NOT NULL,
    pr_updated_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    head_sha text DEFAULT ''::text NOT NULL,
    mergeable_state text,
    additions integer DEFAULT 0 NOT NULL,
    deletions integer DEFAULT 0 NOT NULL,
    changed_files integer DEFAULT 0 NOT NULL,
    CONSTRAINT github_pull_request_state_check CHECK ((state = ANY (ARRAY['open'::text, 'closed'::text, 'merged'::text, 'draft'::text])))
);


--
-- Name: github_pull_request_check_suite; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.github_pull_request_check_suite (
    pr_id uuid NOT NULL,
    suite_id bigint NOT NULL,
    head_sha text NOT NULL,
    app_id bigint NOT NULL,
    conclusion text,
    status text NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: inbox_item; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inbox_item (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    recipient_type text NOT NULL,
    recipient_id uuid NOT NULL,
    type text NOT NULL,
    severity text DEFAULT 'info'::text NOT NULL,
    issue_id uuid,
    title text NOT NULL,
    body text,
    read boolean DEFAULT false NOT NULL,
    archived boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    actor_type text,
    actor_id uuid,
    details jsonb DEFAULT '{}'::jsonb,
    CONSTRAINT inbox_item_recipient_type_check CHECK ((recipient_type = ANY (ARRAY['member'::text, 'agent'::text]))),
    CONSTRAINT inbox_item_severity_check CHECK ((severity = ANY (ARRAY['action_required'::text, 'attention'::text, 'info'::text])))
);


--
-- Name: issue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    title text NOT NULL,
    description text,
    status text DEFAULT 'backlog'::text NOT NULL,
    priority text DEFAULT 'none'::text NOT NULL,
    assignee_type text,
    assignee_id uuid,
    creator_type text NOT NULL,
    creator_id uuid NOT NULL,
    parent_issue_id uuid,
    acceptance_criteria jsonb DEFAULT '[]'::jsonb NOT NULL,
    context_refs jsonb DEFAULT '[]'::jsonb NOT NULL,
    "position" double precision DEFAULT 0 NOT NULL,
    due_date timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    number integer DEFAULT 0 NOT NULL,
    feature_id uuid,
    origin_type text,
    origin_id uuid,
    first_executed_at timestamp with time zone,
    start_date timestamp with time zone,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT issue_assignee_type_check CHECK ((assignee_type = ANY (ARRAY['member'::text, 'agent'::text, 'squad'::text]))),
    CONSTRAINT issue_creator_type_check CHECK ((creator_type = ANY (ARRAY['member'::text, 'agent'::text]))),
    CONSTRAINT issue_metadata_is_object CHECK ((jsonb_typeof(metadata) = 'object'::text)),
    CONSTRAINT issue_metadata_size_limit CHECK ((pg_column_size(metadata) <= 8192)),
    CONSTRAINT issue_origin_type_check CHECK ((origin_type = ANY (ARRAY['autopilot'::text, 'quick_create'::text]))),
    CONSTRAINT issue_priority_check CHECK ((priority = ANY (ARRAY['urgent'::text, 'high'::text, 'medium'::text, 'low'::text, 'none'::text]))),
    CONSTRAINT issue_status_check CHECK ((status = ANY (ARRAY['backlog'::text, 'todo'::text, 'in_progress'::text, 'in_review'::text, 'done'::text, 'blocked'::text, 'cancelled'::text])))
);


--
-- Name: issue_dependency; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_dependency (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    issue_id uuid NOT NULL,
    depends_on_issue_id uuid NOT NULL,
    type text NOT NULL,
    CONSTRAINT issue_dependency_type_check CHECK ((type = ANY (ARRAY['blocks'::text, 'blocked_by'::text, 'related'::text])))
);


--
-- Name: issue_label; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_label (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    name text NOT NULL,
    color text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: issue_pull_request; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_pull_request (
    issue_id uuid NOT NULL,
    pull_request_id uuid NOT NULL,
    linked_by_type text,
    linked_by_id uuid,
    linked_at timestamp with time zone DEFAULT now() NOT NULL,
    close_intent boolean DEFAULT false NOT NULL
);


--
-- Name: issue_reaction; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_reaction (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    issue_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    actor_type text NOT NULL,
    actor_id uuid NOT NULL,
    emoji text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT issue_reaction_actor_type_check CHECK ((actor_type = ANY (ARRAY['member'::text, 'agent'::text])))
);


--
-- Name: issue_subscriber; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_subscriber (
    issue_id uuid NOT NULL,
    user_type text NOT NULL,
    user_id uuid NOT NULL,
    reason text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT issue_subscriber_reason_check CHECK ((reason = ANY (ARRAY['creator'::text, 'assignee'::text, 'commenter'::text, 'mentioned'::text, 'manual'::text]))),
    CONSTRAINT issue_subscriber_user_type_check CHECK ((user_type = ANY (ARRAY['member'::text, 'agent'::text])))
);


--
-- Name: issue_to_label; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.issue_to_label (
    issue_id uuid NOT NULL,
    label_id uuid NOT NULL
);


--
-- Name: member; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.member (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT member_role_check CHECK ((role = ANY (ARRAY['owner'::text, 'admin'::text, 'member'::text])))
);


--
-- Name: notification_preference; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notification_preference (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    user_id uuid NOT NULL,
    preferences jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: pinned_item; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pinned_item (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    user_id uuid NOT NULL,
    item_type text NOT NULL,
    item_id uuid NOT NULL,
    "position" double precision DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT pinned_item_item_type_check CHECK ((item_type = ANY (ARRAY['issue'::text, 'feature'::text])))
);


--
-- Name: feature; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feature (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    title text NOT NULL,
    description text,
    icon text,
    status text DEFAULT 'planned'::text NOT NULL,
    lead_type text,
    lead_id uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    priority text DEFAULT 'none'::text NOT NULL,
    target_branch text,
    CONSTRAINT feature_lead_type_check CHECK ((lead_type = ANY (ARRAY['member'::text, 'agent'::text]))),
    CONSTRAINT feature_priority_check CHECK ((priority = ANY (ARRAY['urgent'::text, 'high'::text, 'medium'::text, 'low'::text, 'none'::text]))),
    CONSTRAINT feature_status_check CHECK ((status = ANY (ARRAY['planned'::text, 'in_progress'::text, 'paused'::text, 'completed'::text, 'cancelled'::text])))
);


--
-- Name: feature_resource; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feature_resource (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    feature_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    resource_type text NOT NULL,
    resource_ref jsonb NOT NULL,
    label text,
    "position" integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    created_by uuid
);


--
-- Name: skill; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.skill (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    name text NOT NULL,
    description text DEFAULT ''::text NOT NULL,
    content text DEFAULT ''::text NOT NULL,
    config jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: skill_file; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.skill_file (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    skill_id uuid NOT NULL,
    path text NOT NULL,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: squad; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.squad (
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


--
-- Name: squad_member; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.squad_member (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    squad_id uuid NOT NULL,
    member_type text NOT NULL,
    member_id uuid NOT NULL,
    role text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT squad_member_member_type_check CHECK ((member_type = ANY (ARRAY['agent'::text, 'member'::text])))
);


--
-- Name: task_message; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_message (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    seq integer NOT NULL,
    type text NOT NULL,
    tool text,
    content text,
    input jsonb,
    output text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: task_token; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_token (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    token_hash text NOT NULL,
    task_id uuid NOT NULL,
    agent_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    user_id uuid NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: task_usage; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_usage (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    task_id uuid NOT NULL,
    provider text DEFAULT ''::text NOT NULL,
    model text NOT NULL,
    input_tokens bigint DEFAULT 0 NOT NULL,
    output_tokens bigint DEFAULT 0 NOT NULL,
    cache_read_tokens bigint DEFAULT 0 NOT NULL,
    cache_write_tokens bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: task_usage_hourly; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_usage_hourly (
    bucket_hour timestamp with time zone NOT NULL,
    workspace_id uuid NOT NULL,
    runtime_id uuid NOT NULL,
    agent_id uuid NOT NULL,
    feature_id uuid,
    provider text NOT NULL,
    model text NOT NULL,
    input_tokens bigint DEFAULT 0 NOT NULL,
    output_tokens bigint DEFAULT 0 NOT NULL,
    cache_read_tokens bigint DEFAULT 0 NOT NULL,
    cache_write_tokens bigint DEFAULT 0 NOT NULL,
    task_count bigint DEFAULT 0 NOT NULL,
    event_count bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: task_usage_hourly_dirty; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_usage_hourly_dirty (
    bucket_hour timestamp with time zone NOT NULL,
    workspace_id uuid NOT NULL,
    runtime_id uuid NOT NULL,
    agent_id uuid NOT NULL,
    feature_id uuid,
    provider text NOT NULL,
    model text NOT NULL,
    enqueued_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: task_usage_hourly_rollup_state; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_usage_hourly_rollup_state (
    id smallint DEFAULT 1 NOT NULL,
    watermark_at timestamp with time zone DEFAULT '1970-01-01 00:00:00+00'::timestamp with time zone NOT NULL,
    last_run_started_at timestamp with time zone,
    last_run_finished_at timestamp with time zone,
    last_run_rows bigint DEFAULT 0 NOT NULL,
    last_error text,
    CONSTRAINT task_usage_hourly_rollup_state_id_check CHECK ((id = 1))
);


--
-- Name: user; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public."user" (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    avatar_url text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    language character varying(20) DEFAULT NULL::character varying,
    profile_description text DEFAULT ''::text NOT NULL,
    timezone text
);


--
-- Name: webhook_delivery; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhook_delivery (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workspace_id uuid NOT NULL,
    autopilot_id uuid NOT NULL,
    trigger_id uuid NOT NULL,
    provider text NOT NULL,
    event text DEFAULT 'webhook.received'::text NOT NULL,
    dedupe_key text,
    dedupe_source text,
    signature_status text DEFAULT 'not_required'::text NOT NULL,
    status text DEFAULT 'queued'::text NOT NULL,
    attempt_count integer DEFAULT 1 NOT NULL,
    selected_headers jsonb DEFAULT '{}'::jsonb NOT NULL,
    content_type text,
    raw_body bytea,
    response_status integer,
    response_body text,
    autopilot_run_id uuid,
    replayed_from_delivery_id uuid,
    error text,
    received_at timestamp with time zone DEFAULT now() NOT NULL,
    last_attempt_at timestamp with time zone DEFAULT now() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT webhook_delivery_provider_check CHECK ((provider = ANY (ARRAY['generic'::text, 'github'::text]))),
    CONSTRAINT webhook_delivery_signature_status_check CHECK ((signature_status = ANY (ARRAY['not_required'::text, 'valid'::text, 'invalid'::text, 'missing'::text]))),
    CONSTRAINT webhook_delivery_status_check CHECK ((status = ANY (ARRAY['queued'::text, 'dispatched'::text, 'rejected'::text, 'ignored'::text, 'failed'::text])))
);


--
-- Name: workspace; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workspace (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    description text,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    context text,
    repos jsonb DEFAULT '[]'::jsonb NOT NULL,
    issue_prefix text DEFAULT ''::text NOT NULL,
    issue_counter integer DEFAULT 0 NOT NULL
);


--
-- Name: activity_log activity_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activity_log
    ADD CONSTRAINT activity_log_pkey PRIMARY KEY (id);


--
-- Name: agent agent_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent
    ADD CONSTRAINT agent_pkey PRIMARY KEY (id);


--
-- Name: agent_runtime agent_runtime_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_runtime
    ADD CONSTRAINT agent_runtime_pkey PRIMARY KEY (id);


--
-- Name: agent_runtime agent_runtime_workspace_id_daemon_id_provider_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_runtime
    ADD CONSTRAINT agent_runtime_workspace_id_daemon_id_provider_key UNIQUE (workspace_id, daemon_id, provider);


--
-- Name: agent_skill agent_skill_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_skill
    ADD CONSTRAINT agent_skill_pkey PRIMARY KEY (agent_id, skill_id);


--
-- Name: agent_task_queue agent_task_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_task_queue
    ADD CONSTRAINT agent_task_queue_pkey PRIMARY KEY (id);


--
-- Name: agent agent_workspace_name_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent
    ADD CONSTRAINT agent_workspace_name_unique UNIQUE (workspace_id, name);


--
-- Name: attachment attachment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachment
    ADD CONSTRAINT attachment_pkey PRIMARY KEY (id);


--
-- Name: autopilot autopilot_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot
    ADD CONSTRAINT autopilot_pkey PRIMARY KEY (id);


--
-- Name: autopilot_run autopilot_run_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot_run
    ADD CONSTRAINT autopilot_run_pkey PRIMARY KEY (id);


--
-- Name: autopilot_trigger autopilot_trigger_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot_trigger
    ADD CONSTRAINT autopilot_trigger_pkey PRIMARY KEY (id);


--
-- Name: chat_message chat_message_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_message
    ADD CONSTRAINT chat_message_pkey PRIMARY KEY (id);


--
-- Name: chat_session chat_session_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_session
    ADD CONSTRAINT chat_session_pkey PRIMARY KEY (id);


--
-- Name: comment comment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment
    ADD CONSTRAINT comment_pkey PRIMARY KEY (id);


--
-- Name: comment_reaction comment_reaction_comment_id_actor_type_actor_id_emoji_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reaction
    ADD CONSTRAINT comment_reaction_comment_id_actor_type_actor_id_emoji_key UNIQUE (comment_id, actor_type, actor_id, emoji);


--
-- Name: comment_reaction comment_reaction_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reaction
    ADD CONSTRAINT comment_reaction_pkey PRIMARY KEY (id);


--
-- Name: daemon_connection daemon_connection_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_connection
    ADD CONSTRAINT daemon_connection_pkey PRIMARY KEY (id);


--
-- Name: daemon_token daemon_token_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_token
    ADD CONSTRAINT daemon_token_pkey PRIMARY KEY (id);


--
-- Name: github_installation github_installation_installation_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_installation
    ADD CONSTRAINT github_installation_installation_id_key UNIQUE (installation_id);


--
-- Name: github_installation github_installation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_installation
    ADD CONSTRAINT github_installation_pkey PRIMARY KEY (id);


--
-- Name: github_pull_request_check_suite github_pull_request_check_suite_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_pull_request_check_suite
    ADD CONSTRAINT github_pull_request_check_suite_pkey PRIMARY KEY (pr_id, suite_id);


--
-- Name: github_pull_request github_pull_request_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_pull_request
    ADD CONSTRAINT github_pull_request_pkey PRIMARY KEY (id);


--
-- Name: github_pull_request github_pull_request_workspace_id_repo_owner_repo_name_pr_nu_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_pull_request
    ADD CONSTRAINT github_pull_request_workspace_id_repo_owner_repo_name_pr_nu_key UNIQUE (workspace_id, repo_owner, repo_name, pr_number);


--
-- Name: inbox_item inbox_item_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inbox_item
    ADD CONSTRAINT inbox_item_pkey PRIMARY KEY (id);


--
-- Name: issue_dependency issue_dependency_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_dependency
    ADD CONSTRAINT issue_dependency_pkey PRIMARY KEY (id);


--
-- Name: issue_label issue_label_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_label
    ADD CONSTRAINT issue_label_pkey PRIMARY KEY (id);


--
-- Name: issue issue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_pkey PRIMARY KEY (id);


--
-- Name: issue_pull_request issue_pull_request_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_pull_request
    ADD CONSTRAINT issue_pull_request_pkey PRIMARY KEY (issue_id, pull_request_id);


--
-- Name: issue_reaction issue_reaction_issue_id_actor_type_actor_id_emoji_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reaction
    ADD CONSTRAINT issue_reaction_issue_id_actor_type_actor_id_emoji_key UNIQUE (issue_id, actor_type, actor_id, emoji);


--
-- Name: issue_reaction issue_reaction_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reaction
    ADD CONSTRAINT issue_reaction_pkey PRIMARY KEY (id);


--
-- Name: issue_subscriber issue_subscriber_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_pkey PRIMARY KEY (issue_id, user_type, user_id);


--
-- Name: issue_to_label issue_to_label_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_to_label
    ADD CONSTRAINT issue_to_label_pkey PRIMARY KEY (issue_id, label_id);


--
-- Name: member member_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_pkey PRIMARY KEY (id);


--
-- Name: member member_workspace_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_workspace_id_user_id_key UNIQUE (workspace_id, user_id);


--
-- Name: notification_preference notification_preference_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_preference
    ADD CONSTRAINT notification_preference_pkey PRIMARY KEY (id);


--
-- Name: notification_preference notification_preference_workspace_id_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_preference
    ADD CONSTRAINT notification_preference_workspace_id_user_id_key UNIQUE (workspace_id, user_id);


--
-- Name: pinned_item pinned_item_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pinned_item
    ADD CONSTRAINT pinned_item_pkey PRIMARY KEY (id);


--
-- Name: pinned_item pinned_item_workspace_id_user_id_item_type_item_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pinned_item
    ADD CONSTRAINT pinned_item_workspace_id_user_id_item_type_item_id_key UNIQUE (workspace_id, user_id, item_type, item_id);


--
-- Name: project feature_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature
    ADD CONSTRAINT feature_pkey PRIMARY KEY (id);


--
-- Name: feature_resource feature_resource_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_resource
    ADD CONSTRAINT feature_resource_pkey PRIMARY KEY (id);


--
-- Name: feature_resource feature_resource_feature_id_resource_type_resource_ref_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_resource
    ADD CONSTRAINT feature_resource_feature_id_resource_type_resource_ref_key UNIQUE (feature_id, resource_type, resource_ref);


--
-- Name: skill_file skill_file_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.skill_file
    ADD CONSTRAINT skill_file_pkey PRIMARY KEY (id);


--
-- Name: skill_file skill_file_skill_id_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.skill_file
    ADD CONSTRAINT skill_file_skill_id_path_key UNIQUE (skill_id, path);


--
-- Name: skill skill_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.skill
    ADD CONSTRAINT skill_pkey PRIMARY KEY (id);


--
-- Name: skill skill_workspace_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.skill
    ADD CONSTRAINT skill_workspace_id_name_key UNIQUE (workspace_id, name);


--
-- Name: squad_member squad_member_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.squad_member
    ADD CONSTRAINT squad_member_pkey PRIMARY KEY (id);


--
-- Name: squad_member squad_member_squad_id_member_type_member_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.squad_member
    ADD CONSTRAINT squad_member_squad_id_member_type_member_id_key UNIQUE (squad_id, member_type, member_id);


--
-- Name: squad squad_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.squad
    ADD CONSTRAINT squad_pkey PRIMARY KEY (id);


--
-- Name: task_message task_message_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_message
    ADD CONSTRAINT task_message_pkey PRIMARY KEY (id);


--
-- Name: task_token task_token_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_token
    ADD CONSTRAINT task_token_pkey PRIMARY KEY (id);


--
-- Name: task_usage_hourly_rollup_state task_usage_hourly_rollup_state_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_usage_hourly_rollup_state
    ADD CONSTRAINT task_usage_hourly_rollup_state_pkey PRIMARY KEY (id);


--
-- Name: task_usage task_usage_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_usage
    ADD CONSTRAINT task_usage_pkey PRIMARY KEY (id);


--
-- Name: task_usage task_usage_task_id_provider_model_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_usage
    ADD CONSTRAINT task_usage_task_id_provider_model_key UNIQUE (task_id, provider, model);


--
-- Name: daemon_connection uq_daemon_agent; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_connection
    ADD CONSTRAINT uq_daemon_agent UNIQUE (agent_id, daemon_id);


--
-- Name: issue uq_issue_workspace_number; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT uq_issue_workspace_number UNIQUE (workspace_id, number);


--
-- Name: task_usage_hourly_dirty uq_task_usage_hourly_dirty_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_usage_hourly_dirty
    ADD CONSTRAINT uq_task_usage_hourly_dirty_key UNIQUE NULLS NOT DISTINCT (bucket_hour, workspace_id, runtime_id, agent_id, feature_id, provider, model);


--
-- Name: task_usage_hourly uq_task_usage_hourly_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_usage_hourly
    ADD CONSTRAINT uq_task_usage_hourly_key UNIQUE NULLS NOT DISTINCT (bucket_hour, workspace_id, runtime_id, agent_id, feature_id, provider, model);


--
-- Name: user user_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."user"
    ADD CONSTRAINT user_email_key UNIQUE (email);


--
-- Name: user user_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."user"
    ADD CONSTRAINT user_pkey PRIMARY KEY (id);


--
-- Name: webhook_delivery webhook_delivery_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_delivery
    ADD CONSTRAINT webhook_delivery_pkey PRIMARY KEY (id);


--
-- Name: workspace workspace_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace
    ADD CONSTRAINT workspace_pkey PRIMARY KEY (id);


--
-- Name: workspace workspace_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workspace
    ADD CONSTRAINT workspace_slug_key UNIQUE (slug);


--
-- Name: comment_issue_resolved_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX comment_issue_resolved_at_idx ON public.comment USING btree (issue_id, resolved_at);


--
-- Name: idx_activity_log_issue_keyset; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_activity_log_issue_keyset ON public.activity_log USING btree (issue_id, created_at DESC, id DESC);


--
-- Name: idx_activity_log_squad_no_action_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_activity_log_squad_no_action_task ON public.activity_log USING btree (issue_id, actor_id, ((details ->> 'task_id'::text))) WHERE ((actor_type = 'agent'::text) AND (action = 'squad_leader_evaluated'::text) AND ((details ->> 'outcome'::text) = 'no_action'::text));


--
-- Name: idx_agent_runtime_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_runtime_status ON public.agent_runtime USING btree (workspace_id, status);


--
-- Name: idx_agent_runtime_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_runtime_workspace ON public.agent_runtime USING btree (workspace_id);


--
-- Name: idx_agent_skill_agent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_skill_agent ON public.agent_skill USING btree (agent_id);


--
-- Name: idx_agent_skill_skill; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_skill_skill ON public.agent_skill USING btree (skill_id);


--
-- Name: idx_agent_task_queue_agent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_task_queue_agent ON public.agent_task_queue USING btree (agent_id, status);


--
-- Name: idx_agent_task_queue_chat_pending; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_task_queue_chat_pending ON public.agent_task_queue USING btree (chat_session_id, created_at DESC) WHERE ((chat_session_id IS NOT NULL) AND (status = ANY (ARRAY['queued'::text, 'dispatched'::text, 'running'::text])));


--
-- Name: idx_agent_task_queue_claim_candidates; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_task_queue_claim_candidates ON public.agent_task_queue USING btree (runtime_id, priority DESC, created_at) WHERE (status = 'queued'::text);


--
-- Name: idx_agent_task_queue_issue_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_task_queue_issue_id ON public.agent_task_queue USING btree (issue_id);


--
-- Name: idx_agent_task_queue_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_task_queue_parent ON public.agent_task_queue USING btree (parent_task_id);


--
-- Name: idx_agent_task_queue_pending; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_task_queue_pending ON public.agent_task_queue USING btree (agent_id, priority DESC, created_at) WHERE (status = ANY (ARRAY['queued'::text, 'dispatched'::text]));


--
-- Name: idx_agent_task_queue_queued_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_task_queue_queued_created_at ON public.agent_task_queue USING btree (created_at) WHERE (status = 'queued'::text);


--
-- Name: idx_agent_task_queue_runtime_pending; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_task_queue_runtime_pending ON public.agent_task_queue USING btree (runtime_id, priority DESC, created_at) WHERE (status = ANY (ARRAY['queued'::text, 'dispatched'::text]));


--
-- Name: idx_agent_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_workspace ON public.agent USING btree (workspace_id);


--
-- Name: idx_attachment_chat_message; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_attachment_chat_message ON public.attachment USING btree (chat_message_id) WHERE (chat_message_id IS NOT NULL);


--
-- Name: idx_attachment_chat_session; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_attachment_chat_session ON public.attachment USING btree (chat_session_id) WHERE (chat_session_id IS NOT NULL);


--
-- Name: idx_attachment_comment; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_attachment_comment ON public.attachment USING btree (comment_id) WHERE (comment_id IS NOT NULL);


--
-- Name: idx_attachment_issue; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_attachment_issue ON public.attachment USING btree (issue_id) WHERE (issue_id IS NOT NULL);


--
-- Name: idx_attachment_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_attachment_workspace ON public.attachment USING btree (workspace_id);


--
-- Name: idx_autopilot_assignee; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_autopilot_assignee ON public.autopilot USING btree (assignee_id);


--
-- Name: idx_autopilot_assignee_type_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_autopilot_assignee_type_id ON public.autopilot USING btree (assignee_type, assignee_id);


--
-- Name: idx_autopilot_feature; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_autopilot_feature ON public.autopilot USING btree (feature_id);


--
-- Name: idx_autopilot_run_autopilot; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_autopilot_run_autopilot ON public.autopilot_run USING btree (autopilot_id, created_at DESC);


--
-- Name: idx_autopilot_run_issue; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_autopilot_run_issue ON public.autopilot_run USING btree (issue_id) WHERE (issue_id IS NOT NULL);


--
-- Name: idx_autopilot_run_squad_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_autopilot_run_squad_id ON public.autopilot_run USING btree (squad_id) WHERE (squad_id IS NOT NULL);


--
-- Name: idx_autopilot_run_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_autopilot_run_status ON public.autopilot_run USING btree (autopilot_id, status) WHERE (status = ANY (ARRAY['issue_created'::text, 'running'::text]));


--
-- Name: idx_autopilot_trigger_autopilot; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_autopilot_trigger_autopilot ON public.autopilot_trigger USING btree (autopilot_id);


--
-- Name: idx_autopilot_trigger_next_run; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_autopilot_trigger_next_run ON public.autopilot_trigger USING btree (next_run_at) WHERE ((enabled = true) AND (kind = 'schedule'::text));


--
-- Name: idx_autopilot_trigger_webhook_token; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_autopilot_trigger_webhook_token ON public.autopilot_trigger USING btree (webhook_token) WHERE ((kind = 'webhook'::text) AND (webhook_token IS NOT NULL));


--
-- Name: idx_autopilot_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_autopilot_workspace ON public.autopilot USING btree (workspace_id);


--
-- Name: idx_chat_message_session; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_message_session ON public.chat_message USING btree (chat_session_id, created_at);


--
-- Name: idx_chat_session_creator; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_session_creator ON public.chat_session USING btree (creator_id, workspace_id);


--
-- Name: idx_chat_session_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_session_workspace ON public.chat_session USING btree (workspace_id);


--
-- Name: idx_comment_issue_keyset; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_comment_issue_keyset ON public.comment USING btree (issue_id, created_at DESC, id DESC);


--
-- Name: idx_comment_reaction_comment_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_comment_reaction_comment_id ON public.comment_reaction USING btree (comment_id);


--
-- Name: idx_daemon_token_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_daemon_token_hash ON public.daemon_token USING btree (token_hash);


--
-- Name: idx_daemon_token_workspace_daemon; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_daemon_token_workspace_daemon ON public.daemon_token USING btree (workspace_id, daemon_id);


--
-- Name: idx_github_installation_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_github_installation_workspace ON public.github_installation USING btree (workspace_id);


--
-- Name: idx_github_pr_check_suite_aggregate; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_github_pr_check_suite_aggregate ON public.github_pull_request_check_suite USING btree (pr_id, head_sha, app_id, updated_at DESC);


--
-- Name: idx_github_pull_request_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_github_pull_request_workspace ON public.github_pull_request USING btree (workspace_id);


--
-- Name: idx_inbox_recipient; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_inbox_recipient ON public.inbox_item USING btree (recipient_type, recipient_id, read);


--
-- Name: idx_issue_assignee; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_assignee ON public.issue USING btree (assignee_type, assignee_id);


--
-- Name: idx_issue_first_executed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_first_executed_at ON public.issue USING btree (workspace_id, first_executed_at) WHERE (first_executed_at IS NOT NULL);


--
-- Name: idx_issue_metadata_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_metadata_gin ON public.issue USING gin (metadata jsonb_path_ops);


--
-- Name: idx_issue_origin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_origin ON public.issue USING btree (origin_type, origin_id) WHERE (origin_type IS NOT NULL);


--
-- Name: idx_issue_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_parent ON public.issue USING btree (parent_issue_id);


--
-- Name: idx_issue_feature; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_feature ON public.issue USING btree (feature_id);


--
-- Name: idx_issue_pull_request_pr; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_pull_request_pr ON public.issue_pull_request USING btree (pull_request_id);


--
-- Name: idx_issue_reaction_issue_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_reaction_issue_id ON public.issue_reaction USING btree (issue_id);


--
-- Name: idx_issue_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_status ON public.issue USING btree (workspace_id, status);


--
-- Name: idx_issue_subscriber_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_subscriber_user ON public.issue_subscriber USING btree (user_type, user_id);


--
-- Name: idx_issue_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_workspace ON public.issue USING btree (workspace_id);


--
-- Name: idx_issue_workspace_number; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_issue_workspace_number ON public.issue USING btree (workspace_id, number);


--
-- Name: idx_member_user_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_member_user_workspace ON public.member USING btree (user_id, workspace_id);


--
-- Name: idx_member_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_member_workspace ON public.member USING btree (workspace_id);


--
-- Name: idx_one_pending_task_per_issue_agent; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_one_pending_task_per_issue_agent ON public.agent_task_queue USING btree (issue_id, agent_id) WHERE (status = ANY (ARRAY['queued'::text, 'dispatched'::text]));


--
-- Name: idx_pinned_item_user_ws; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_pinned_item_user_ws ON public.pinned_item USING btree (workspace_id, user_id, "position");


--
-- Name: idx_feature_resource_feature; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feature_resource_feature ON public.feature_resource USING btree (feature_id, "position");


--
-- Name: idx_feature_resource_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feature_resource_workspace ON public.feature_resource USING btree (workspace_id);


--
-- Name: idx_feature_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feature_workspace ON public.feature USING btree (workspace_id);


--
-- Name: idx_skill_file_skill; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_skill_file_skill ON public.skill_file USING btree (skill_id);


--
-- Name: idx_skill_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_skill_workspace ON public.skill USING btree (workspace_id);


--
-- Name: idx_squad_member_entity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_squad_member_entity ON public.squad_member USING btree (member_type, member_id);


--
-- Name: idx_squad_member_squad; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_squad_member_squad ON public.squad_member USING btree (squad_id);


--
-- Name: idx_squad_workspace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_squad_workspace ON public.squad USING btree (workspace_id);


--
-- Name: idx_task_message_task_id_seq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_message_task_id_seq ON public.task_message USING btree (task_id, seq);


--
-- Name: idx_task_token_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_task_token_hash ON public.task_token USING btree (token_hash);


--
-- Name: idx_task_token_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_token_task ON public.task_token USING btree (task_id);


--
-- Name: idx_task_usage_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_usage_created_at ON public.task_usage USING btree (created_at);


--
-- Name: idx_task_usage_created_at_legacy; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_usage_created_at_legacy ON public.task_usage USING btree (created_at) WHERE (updated_at IS NULL);


--
-- Name: idx_task_usage_hourly_dirty_enqueued_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_usage_hourly_dirty_enqueued_at ON public.task_usage_hourly_dirty USING btree (enqueued_at);


--
-- Name: idx_task_usage_hourly_runtime_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_usage_hourly_runtime_time ON public.task_usage_hourly USING btree (runtime_id, bucket_hour DESC);


--
-- Name: idx_task_usage_hourly_workspace_agent_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_usage_hourly_workspace_agent_time ON public.task_usage_hourly USING btree (workspace_id, agent_id, bucket_hour DESC);


--
-- Name: idx_task_usage_hourly_workspace_feature_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_usage_hourly_workspace_feature_time ON public.task_usage_hourly USING btree (workspace_id, feature_id, bucket_hour DESC) WHERE (feature_id IS NOT NULL);


--
-- Name: idx_task_usage_hourly_workspace_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_usage_hourly_workspace_time ON public.task_usage_hourly USING btree (workspace_id, bucket_hour DESC);


--
-- Name: idx_task_usage_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_usage_task_id ON public.task_usage USING btree (task_id);


--
-- Name: idx_task_usage_updated_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_usage_updated_at ON public.task_usage USING btree (updated_at);


--
-- Name: idx_webhook_delivery_autopilot; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webhook_delivery_autopilot ON public.webhook_delivery USING btree (autopilot_id, created_at DESC);


--
-- Name: idx_webhook_delivery_dedupe; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_webhook_delivery_dedupe ON public.webhook_delivery USING btree (trigger_id, dedupe_key) WHERE ((dedupe_key IS NOT NULL) AND (status <> ALL (ARRAY['rejected'::text, 'failed'::text])));


--
-- Name: idx_webhook_delivery_run; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webhook_delivery_run ON public.webhook_delivery USING btree (autopilot_run_id) WHERE (autopilot_run_id IS NOT NULL);


--
-- Name: issue_label_workspace_name_lower_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX issue_label_workspace_name_lower_idx ON public.issue_label USING btree (workspace_id, lower(name));


--
-- Name: agent_task_queue trg_atq_dirty_hourly; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_atq_dirty_hourly BEFORE DELETE OR UPDATE OF runtime_id, issue_id ON public.agent_task_queue FOR EACH ROW EXECUTE FUNCTION public.enqueue_task_usage_hourly_dirty_for_atq();


--
-- Name: issue trg_issue_delete_dirty_hourly; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_issue_delete_dirty_hourly BEFORE DELETE ON public.issue FOR EACH ROW EXECUTE FUNCTION public.enqueue_task_usage_hourly_dirty_for_issue_delete();


--
-- Name: issue trg_issue_feature_dirty_hourly; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_issue_feature_dirty_hourly BEFORE UPDATE OF feature_id ON public.issue FOR EACH ROW EXECUTE FUNCTION public.enqueue_task_usage_hourly_dirty_for_issue_feature();


--
-- Name: task_usage trg_tu_dirty_hourly; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_tu_dirty_hourly BEFORE DELETE ON public.task_usage FOR EACH ROW EXECUTE FUNCTION public.enqueue_task_usage_hourly_dirty_for_tu();


--
-- Name: activity_log activity_log_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activity_log
    ADD CONSTRAINT activity_log_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: activity_log activity_log_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.activity_log
    ADD CONSTRAINT activity_log_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: agent agent_archived_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent
    ADD CONSTRAINT agent_archived_by_fkey FOREIGN KEY (archived_by) REFERENCES public."user"(id);


--
-- Name: agent agent_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent
    ADD CONSTRAINT agent_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public."user"(id);


--
-- Name: agent agent_runtime_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent
    ADD CONSTRAINT agent_runtime_id_fkey FOREIGN KEY (runtime_id) REFERENCES public.agent_runtime(id) ON DELETE RESTRICT;


--
-- Name: agent_runtime agent_runtime_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_runtime
    ADD CONSTRAINT agent_runtime_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public."user"(id);


--
-- Name: agent_runtime agent_runtime_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_runtime
    ADD CONSTRAINT agent_runtime_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: agent_skill agent_skill_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_skill
    ADD CONSTRAINT agent_skill_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agent(id) ON DELETE CASCADE;


--
-- Name: agent_skill agent_skill_skill_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_skill
    ADD CONSTRAINT agent_skill_skill_id_fkey FOREIGN KEY (skill_id) REFERENCES public.skill(id) ON DELETE CASCADE;


--
-- Name: agent_task_queue agent_task_queue_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_task_queue
    ADD CONSTRAINT agent_task_queue_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agent(id) ON DELETE CASCADE;


--
-- Name: agent_task_queue agent_task_queue_autopilot_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_task_queue
    ADD CONSTRAINT agent_task_queue_autopilot_run_id_fkey FOREIGN KEY (autopilot_run_id) REFERENCES public.autopilot_run(id) ON DELETE SET NULL;


--
-- Name: agent_task_queue agent_task_queue_chat_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_task_queue
    ADD CONSTRAINT agent_task_queue_chat_session_id_fkey FOREIGN KEY (chat_session_id) REFERENCES public.chat_session(id) ON DELETE SET NULL;


--
-- Name: agent_task_queue agent_task_queue_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_task_queue
    ADD CONSTRAINT agent_task_queue_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: agent_task_queue agent_task_queue_parent_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_task_queue
    ADD CONSTRAINT agent_task_queue_parent_task_id_fkey FOREIGN KEY (parent_task_id) REFERENCES public.agent_task_queue(id) ON DELETE SET NULL;


--
-- Name: agent_task_queue agent_task_queue_runtime_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_task_queue
    ADD CONSTRAINT agent_task_queue_runtime_id_fkey FOREIGN KEY (runtime_id) REFERENCES public.agent_runtime(id) ON DELETE CASCADE;


--
-- Name: agent_task_queue agent_task_queue_trigger_comment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_task_queue
    ADD CONSTRAINT agent_task_queue_trigger_comment_id_fkey FOREIGN KEY (trigger_comment_id) REFERENCES public.comment(id) ON DELETE SET NULL;


--
-- Name: agent agent_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent
    ADD CONSTRAINT agent_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: attachment attachment_chat_message_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachment
    ADD CONSTRAINT attachment_chat_message_id_fkey FOREIGN KEY (chat_message_id) REFERENCES public.chat_message(id) ON DELETE CASCADE;


--
-- Name: attachment attachment_chat_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachment
    ADD CONSTRAINT attachment_chat_session_id_fkey FOREIGN KEY (chat_session_id) REFERENCES public.chat_session(id) ON DELETE CASCADE;


--
-- Name: attachment attachment_comment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachment
    ADD CONSTRAINT attachment_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES public.comment(id) ON DELETE CASCADE;


--
-- Name: attachment attachment_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachment
    ADD CONSTRAINT attachment_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: attachment attachment_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachment
    ADD CONSTRAINT attachment_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: autopilot autopilot_feature_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot
    ADD CONSTRAINT autopilot_feature_id_fkey FOREIGN KEY (feature_id) REFERENCES public.feature(id) ON DELETE SET NULL;


--
-- Name: autopilot_run autopilot_run_autopilot_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot_run
    ADD CONSTRAINT autopilot_run_autopilot_id_fkey FOREIGN KEY (autopilot_id) REFERENCES public.autopilot(id) ON DELETE CASCADE;


--
-- Name: autopilot_run autopilot_run_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot_run
    ADD CONSTRAINT autopilot_run_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE SET NULL;


--
-- Name: autopilot_run autopilot_run_squad_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot_run
    ADD CONSTRAINT autopilot_run_squad_id_fkey FOREIGN KEY (squad_id) REFERENCES public.squad(id) ON DELETE SET NULL;


--
-- Name: autopilot_run autopilot_run_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot_run
    ADD CONSTRAINT autopilot_run_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.agent_task_queue(id) ON DELETE SET NULL;


--
-- Name: autopilot_run autopilot_run_trigger_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot_run
    ADD CONSTRAINT autopilot_run_trigger_id_fkey FOREIGN KEY (trigger_id) REFERENCES public.autopilot_trigger(id) ON DELETE SET NULL;


--
-- Name: autopilot_trigger autopilot_trigger_autopilot_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot_trigger
    ADD CONSTRAINT autopilot_trigger_autopilot_id_fkey FOREIGN KEY (autopilot_id) REFERENCES public.autopilot(id) ON DELETE CASCADE;


--
-- Name: autopilot autopilot_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.autopilot
    ADD CONSTRAINT autopilot_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: chat_message chat_message_chat_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_message
    ADD CONSTRAINT chat_message_chat_session_id_fkey FOREIGN KEY (chat_session_id) REFERENCES public.chat_session(id) ON DELETE CASCADE;


--
-- Name: chat_session chat_session_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_session
    ADD CONSTRAINT chat_session_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agent(id) ON DELETE CASCADE;


--
-- Name: chat_session chat_session_creator_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_session
    ADD CONSTRAINT chat_session_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: chat_session chat_session_runtime_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_session
    ADD CONSTRAINT chat_session_runtime_id_fkey FOREIGN KEY (runtime_id) REFERENCES public.agent_runtime(id) ON DELETE SET NULL;


--
-- Name: chat_session chat_session_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_session
    ADD CONSTRAINT chat_session_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: comment comment_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment
    ADD CONSTRAINT comment_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: comment comment_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment
    ADD CONSTRAINT comment_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.comment(id) ON DELETE CASCADE;


--
-- Name: comment_reaction comment_reaction_comment_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reaction
    ADD CONSTRAINT comment_reaction_comment_id_fkey FOREIGN KEY (comment_id) REFERENCES public.comment(id) ON DELETE CASCADE;


--
-- Name: comment_reaction comment_reaction_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment_reaction
    ADD CONSTRAINT comment_reaction_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: comment comment_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.comment
    ADD CONSTRAINT comment_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: daemon_connection daemon_connection_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_connection
    ADD CONSTRAINT daemon_connection_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agent(id) ON DELETE CASCADE;


--
-- Name: daemon_token daemon_token_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.daemon_token
    ADD CONSTRAINT daemon_token_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: github_installation github_installation_connected_by_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_installation
    ADD CONSTRAINT github_installation_connected_by_id_fkey FOREIGN KEY (connected_by_id) REFERENCES public."user"(id) ON DELETE SET NULL;


--
-- Name: github_installation github_installation_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_installation
    ADD CONSTRAINT github_installation_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: github_pull_request_check_suite github_pull_request_check_suite_pr_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_pull_request_check_suite
    ADD CONSTRAINT github_pull_request_check_suite_pr_id_fkey FOREIGN KEY (pr_id) REFERENCES public.github_pull_request(id) ON DELETE CASCADE;


--
-- Name: github_pull_request github_pull_request_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_pull_request
    ADD CONSTRAINT github_pull_request_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: inbox_item inbox_item_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inbox_item
    ADD CONSTRAINT inbox_item_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: inbox_item inbox_item_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inbox_item
    ADD CONSTRAINT inbox_item_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: issue_dependency issue_dependency_depends_on_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_dependency
    ADD CONSTRAINT issue_dependency_depends_on_issue_id_fkey FOREIGN KEY (depends_on_issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: issue_dependency issue_dependency_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_dependency
    ADD CONSTRAINT issue_dependency_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: issue_label issue_label_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_label
    ADD CONSTRAINT issue_label_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: issue issue_parent_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_parent_issue_id_fkey FOREIGN KEY (parent_issue_id) REFERENCES public.issue(id) ON DELETE SET NULL;


--
-- Name: issue issue_feature_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_feature_id_fkey FOREIGN KEY (feature_id) REFERENCES public.feature(id) ON DELETE SET NULL;


--
-- Name: issue_pull_request issue_pull_request_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_pull_request
    ADD CONSTRAINT issue_pull_request_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: issue_pull_request issue_pull_request_pull_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_pull_request
    ADD CONSTRAINT issue_pull_request_pull_request_id_fkey FOREIGN KEY (pull_request_id) REFERENCES public.github_pull_request(id) ON DELETE CASCADE;


--
-- Name: issue_reaction issue_reaction_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reaction
    ADD CONSTRAINT issue_reaction_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: issue_reaction issue_reaction_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_reaction
    ADD CONSTRAINT issue_reaction_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: issue_subscriber issue_subscriber_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_subscriber
    ADD CONSTRAINT issue_subscriber_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: issue_to_label issue_to_label_issue_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_to_label
    ADD CONSTRAINT issue_to_label_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES public.issue(id) ON DELETE CASCADE;


--
-- Name: issue_to_label issue_to_label_label_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue_to_label
    ADD CONSTRAINT issue_to_label_label_id_fkey FOREIGN KEY (label_id) REFERENCES public.issue_label(id) ON DELETE CASCADE;


--
-- Name: issue issue_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.issue
    ADD CONSTRAINT issue_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: member member_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: member member_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.member
    ADD CONSTRAINT member_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: notification_preference notification_preference_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_preference
    ADD CONSTRAINT notification_preference_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: notification_preference notification_preference_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_preference
    ADD CONSTRAINT notification_preference_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: pinned_item pinned_item_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pinned_item
    ADD CONSTRAINT pinned_item_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: pinned_item pinned_item_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pinned_item
    ADD CONSTRAINT pinned_item_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: feature_resource feature_resource_feature_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_resource
    ADD CONSTRAINT feature_resource_feature_id_fkey FOREIGN KEY (feature_id) REFERENCES public.feature(id) ON DELETE CASCADE;


--
-- Name: feature_resource feature_resource_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_resource
    ADD CONSTRAINT feature_resource_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: project feature_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature
    ADD CONSTRAINT feature_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: skill skill_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.skill
    ADD CONSTRAINT skill_created_by_fkey FOREIGN KEY (created_by) REFERENCES public."user"(id);


--
-- Name: skill_file skill_file_skill_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.skill_file
    ADD CONSTRAINT skill_file_skill_id_fkey FOREIGN KEY (skill_id) REFERENCES public.skill(id) ON DELETE CASCADE;


--
-- Name: skill skill_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.skill
    ADD CONSTRAINT skill_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: squad squad_leader_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.squad
    ADD CONSTRAINT squad_leader_id_fkey FOREIGN KEY (leader_id) REFERENCES public.agent(id) ON DELETE RESTRICT;


--
-- Name: squad_member squad_member_squad_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.squad_member
    ADD CONSTRAINT squad_member_squad_id_fkey FOREIGN KEY (squad_id) REFERENCES public.squad(id) ON DELETE CASCADE;


--
-- Name: squad squad_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.squad
    ADD CONSTRAINT squad_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: task_message task_message_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_message
    ADD CONSTRAINT task_message_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.agent_task_queue(id) ON DELETE CASCADE;


--
-- Name: task_token task_token_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_token
    ADD CONSTRAINT task_token_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agent(id) ON DELETE CASCADE;


--
-- Name: task_token task_token_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_token
    ADD CONSTRAINT task_token_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.agent_task_queue(id) ON DELETE CASCADE;


--
-- Name: task_token task_token_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_token
    ADD CONSTRAINT task_token_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."user"(id) ON DELETE CASCADE;


--
-- Name: task_token task_token_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_token
    ADD CONSTRAINT task_token_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;


--
-- Name: task_usage task_usage_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_usage
    ADD CONSTRAINT task_usage_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.agent_task_queue(id) ON DELETE CASCADE;


--
-- Name: webhook_delivery webhook_delivery_autopilot_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_delivery
    ADD CONSTRAINT webhook_delivery_autopilot_id_fkey FOREIGN KEY (autopilot_id) REFERENCES public.autopilot(id) ON DELETE CASCADE;


--
-- Name: webhook_delivery webhook_delivery_autopilot_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_delivery
    ADD CONSTRAINT webhook_delivery_autopilot_run_id_fkey FOREIGN KEY (autopilot_run_id) REFERENCES public.autopilot_run(id) ON DELETE SET NULL;


--
-- Name: webhook_delivery webhook_delivery_replayed_from_delivery_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_delivery
    ADD CONSTRAINT webhook_delivery_replayed_from_delivery_id_fkey FOREIGN KEY (replayed_from_delivery_id) REFERENCES public.webhook_delivery(id) ON DELETE SET NULL;


--
-- Name: webhook_delivery webhook_delivery_trigger_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_delivery
    ADD CONSTRAINT webhook_delivery_trigger_id_fkey FOREIGN KEY (trigger_id) REFERENCES public.autopilot_trigger(id) ON DELETE CASCADE;


--
-- Name: webhook_delivery webhook_delivery_workspace_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhook_delivery
    ADD CONSTRAINT webhook_delivery_workspace_id_fkey FOREIGN KEY (workspace_id) REFERENCES public.workspace(id) ON DELETE CASCADE;

-- Seed the singleton rollup-state row consumed by rollup_task_usage_hourly().
-- The table CHECK constraint enforces id = 1.
INSERT INTO task_usage_hourly_rollup_state (id) VALUES (1) ON CONFLICT DO NOTHING;

-- Singleton implicit user for personal-fork single-user model. All authenticated
-- requests are attributed to this row; foreign keys on author_id, owner_id etc.
-- continue to work without a login surface. Idempotent.
INSERT INTO "user" (id, name, email)
VALUES ('00000000-0000-0000-0000-000000000001', 'You', 'local@multica')
ON CONFLICT (id) DO NOTHING;
