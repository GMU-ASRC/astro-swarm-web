from app_settings import (
    ASSAULT_SWEEP_TRIALS,
    assault_sweep_max,
    get_enemy_start,
    get_seed,
    get_sweep_max,
    get_sweep_trials,
    is_assault_level,
)
from config import Config


def sweep_params(level_id):
    if is_assault_level(level_id):
        return assault_sweep_max(level_id), ASSAULT_SWEEP_TRIALS
    return get_sweep_max(), get_sweep_trials()


def config_for(evaluation):
    enemy_x, enemy_y = get_enemy_start()
    sweep_max, sweep_trials = sweep_params(evaluation.level_id)
    return {
        "seed": get_seed(),
        "sweep_max": sweep_max,
        "sweep_trials": sweep_trials,
        "match_seconds": Config.EVAL_MATCH_CAP_SECONDS,
        "goal_tail_seconds": Config.EVAL_GOAL_TAIL_SECONDS,
        "enemy_x": enemy_x,
        "enemy_y": enemy_y,
        "sweep_spawn": Config.EVAL_SWEEP_SPAWN,
        "level_id": evaluation.level_id or "farp",
        "collisions": bool(evaluation.collisions),
    }
