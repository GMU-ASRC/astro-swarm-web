from migrator.versions.v001_legacy_schema import LegacySchema
from migrator.versions.v002_evaluation_indexes import EvaluationIndexes

MIGRATIONS = [
    LegacySchema(),
    EvaluationIndexes(),
]
