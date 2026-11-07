import legacy_copy
import legacy_verify
from database import db
from migrator.migration import Migration


class CopyToBenchmarks(Migration):
    version = 4
    name = "copy_to_benchmarks"
    changes_data = True

    def describe(self, context):
        try:
            row_count = "%d" % legacy_copy.pending_row_count()
        except Exception:
            db.session.rollback()
            row_count = "every"
        return [
            "copy %s player_evaluations row(s) into benchmarks and benchmark_runs" % row_count,
            "write replay frames kept inside the database out to replays/<evaluation id>/inline/",
            "leave the results and replays columns untouched",
            "verify every row, run and inline frame against the original data",
        ]

    def run(self, context):
        context.log("%d row(s) to copy", legacy_copy.pending_row_count())
        copied = legacy_copy.copy_all_pending(log=context.log)
        context.log("copied %d row(s)", copied)

    def verify(self, context):
        problems = legacy_verify.verify_all(log=context.log)
        remaining = legacy_copy.pending_row_count()
        if remaining:
            problems.append("%d row(s) were not copied" % remaining)
        return problems
