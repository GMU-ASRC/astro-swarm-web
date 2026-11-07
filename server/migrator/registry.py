from migrator.versions.v001_legacy_schema import LegacySchema
from migrator.versions.v002_evaluation_indexes import EvaluationIndexes
from migrator.versions.v003_benchmark_tables import BenchmarkTables
from migrator.versions.v004_copy_to_benchmarks import CopyToBenchmarks

MIGRATIONS = [
    LegacySchema(),
    EvaluationIndexes(),
    BenchmarkTables(),
    CopyToBenchmarks(),
]
