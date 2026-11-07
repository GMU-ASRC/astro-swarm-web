from sqlalchemy import Float, cast, func
from sqlalchemy.orm import defer

from database import db
from models import PlayerEvaluation

RATE_KEYS = ("success_rate", "attacker_rate", "defender_rate")

LARGE_COLUMNS = (
    PlayerEvaluation.replays,
    PlayerEvaluation.results,
    PlayerEvaluation.algorithm,
    PlayerEvaluation.placements,
    PlayerEvaluation.genome_schedule,
)


def results_number(key):
    return cast(func.json_extract_path_text(PlayerEvaluation.results, key), Float)


def light_evaluations_query():
    rate_columns = [results_number(key).label(key) for key in RATE_KEYS]
    large_column_options = [defer(column) for column in LARGE_COLUMNS]
    return db.session.query(PlayerEvaluation, *rate_columns).options(*large_column_options)


def with_light_rates(rows):
    evaluations = []
    for evaluation, *rate_values in rows:
        evaluation.light_rates = dict(zip(RATE_KEYS, rate_values))
        evaluations.append(evaluation)
    return evaluations


def light_evaluations(query):
    return with_light_rates(query.all())


def success_rate_of(evaluation):
    return evaluation.light_rates.get("success_rate")
