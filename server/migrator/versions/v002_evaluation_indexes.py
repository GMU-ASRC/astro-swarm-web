from migrator.migration import Migration

INDEXES = {
    "ix_player_evaluations_level_status": "ON player_evaluations (level_id, status)",
    "ix_player_evaluations_player_created": "ON player_evaluations (player_id, created_at DESC)",
    "ix_player_evaluations_queued": "ON player_evaluations (created_at) WHERE status = 'queued'",
    "ix_player_evaluations_running": "ON player_evaluations (last_update) WHERE status = 'running'",
}

INVALID_INDEX_QUERY = """
SELECT index_class.relname
FROM pg_index
JOIN pg_class index_class ON index_class.oid = pg_index.indexrelid
WHERE index_class.relname = :name AND NOT pg_index.indisvalid
"""

VALID_INDEX_QUERY = """
SELECT EXISTS (
    SELECT 1
    FROM pg_index
    JOIN pg_class index_class ON index_class.oid = pg_index.indexrelid
    WHERE index_class.relname = :name AND pg_index.indisvalid
)
"""


class EvaluationIndexes(Migration):
    version = 2
    name = "evaluation_indexes"
    changes_data = False

    def describe(self, context):
        return ["create index %s %s" % (name, definition) for name, definition in INDEXES.items()]

    def run(self, context):
        for name, definition in INDEXES.items():
            if context.rows(INVALID_INDEX_QUERY, {"name": name}):
                context.log("dropping invalid index %s from an earlier failed build", name)
                context.execute_outside_transaction("DROP INDEX CONCURRENTLY IF EXISTS %s" % name)
            context.log("building index %s", name)
            context.execute_outside_transaction("CREATE INDEX CONCURRENTLY IF NOT EXISTS %s %s" % (name, definition))

    def verify(self, context):
        return [
            "index %s is missing or invalid" % name
            for name in INDEXES
            if not context.scalar(VALID_INDEX_QUERY, {"name": name})
        ]
