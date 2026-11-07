CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

\echo 'Table sizes (total, heap, toast and indexes)'
SELECT
    relname AS table_name,
    pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
    pg_size_pretty(pg_relation_size(relid)) AS heap_size,
    pg_size_pretty(pg_total_relation_size(relid) - pg_relation_size(relid) - pg_indexes_size(relid)) AS toast_size,
    pg_size_pretty(pg_indexes_size(relid)) AS index_size,
    n_live_tup AS live_rows,
    n_dead_tup AS dead_rows
FROM pg_catalog.pg_stat_user_tables
ORDER BY pg_total_relation_size(relid) DESC;

\echo 'Evaluations by status'
SELECT status, count(*) FROM player_evaluations GROUP BY status ORDER BY count(*) DESC;

\echo 'Average and largest stored JSON per evaluation'
SELECT
    pg_size_pretty(avg(pg_column_size(results))::bigint) AS average_results,
    pg_size_pretty(max(pg_column_size(results))::bigint) AS largest_results,
    pg_size_pretty(avg(pg_column_size(replays))::bigint) AS average_replays,
    pg_size_pretty(max(pg_column_size(replays))::bigint) AS largest_replays,
    pg_size_pretty(sum(pg_column_size(replays))::bigint) AS all_replays
FROM player_evaluations;

\echo 'Evaluations that still keep replay frames inside the database'
SELECT count(*) AS rows_with_inline_frames
FROM player_evaluations
WHERE replays::text LIKE '%"frames_packed"%' OR replays::text LIKE '%"frames": [[%';

\echo 'Slowest queries (needs pg_stat_statements)'
SELECT
    round(mean_exec_time::numeric, 1) AS mean_ms,
    calls,
    round(total_exec_time::numeric / 1000, 1) AS total_seconds,
    left(regexp_replace(query, '\s+', ' ', 'g'), 160) AS query
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
