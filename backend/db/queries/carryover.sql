-- name: ComputeActualMinutesForTask :one
-- Sums duration of non-skipped blocks referencing a given task_id across all accepted plans.
SELECT COALESCE(SUM((elem->>'duration')::int), 0)::int AS total_minutes
FROM day_plans, jsonb_array_elements(blocks) AS elem
WHERE status = 'accepted'
  AND elem->>'task_id' = @task_id::text
  AND (elem->>'skipped')::boolean = false;

-- name: GetSkippedBlocksFromLastPlan :many
-- Returns skipped blocks with task_ids from the most recent accepted plan before today.
SELECT elem->>'id' AS block_id,
       elem->>'title' AS title,
       elem->>'category' AS category,
       (elem->>'duration')::int AS duration,
       (elem->>'task_id') AS task_id
FROM day_plans, jsonb_array_elements(blocks) AS elem
WHERE status = 'accepted'
  AND plan_date < @today::date
  AND (elem->>'skipped')::boolean = true
  AND elem->>'task_id' IS NOT NULL
  AND elem->>'task_id' != 'null'
  AND elem->>'task_id' != ''
ORDER BY plan_date DESC;
