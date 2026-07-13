from config import Config
from database import db
from models import AppSetting

ENEMY_X_KEY = "enemy_start_x"
ENEMY_Y_KEY = "enemy_start_y"
SWEEP_MAX_KEY = "sweep_max"
SWEEP_TRIALS_KEY = "sweep_trials"

SWEEP_SEED_OFFSET = 100000
SWEEP_SEED_STRIDE = 1000000
SWEEP_MATCH_OFFSET = 500000

LEVELS = [
    {"id": "farp1", "name": "Level 1 - Defense (Place)"},
    {"id": "farp2", "name": "Level 2 - Defense (Ring)"},
]

# Levels whose entries are player-piloted runs: the worker renders the recorded
# run into a replay instead of simulating anything.
PILOT_LEVELS = [
    {"id": "farp3", "name": "Level 3 - Evasion (Pilot)"},
]

LEVEL_ALIASES = {
    "farp1": ["farp1", "farp"],
}


def level_ids_for(level_id):
    return LEVEL_ALIASES.get(level_id, [level_id])


def is_benchmark_level(level_id):
    return any(level["id"] == level_id for level in LEVELS)


def is_pilot_level(level_id):
    return any(level["id"] == level_id for level in PILOT_LEVELS)


def _get(key):
    setting = db.session.get(AppSetting, key)
    if setting is None:
        return None
    return setting.value


def _set(key, value):
    setting = db.session.get(AppSetting, key)
    if setting is None:
        db.session.add(AppSetting(key=key, value=str(value)))
    else:
        setting.value = str(value)
    db.session.commit()


def get_enemy_start():
    def read(key, default):
        value = _get(key)
        if value is None:
            return default
        try:
            return float(value)
        except (TypeError, ValueError):
            return default

    return read(ENEMY_X_KEY, Config.EVAL_ENEMY_X), read(ENEMY_Y_KEY, Config.EVAL_ENEMY_Y)


def set_enemy_start(x, y):
    x = float(x)
    y = float(y)
    _set(ENEMY_X_KEY, x)
    _set(ENEMY_Y_KEY, y)
    return x, y


def _read_int(key, default, low, high):
    value = _get(key)
    if value is None:
        return default
    try:
        return max(low, min(high, int(value)))
    except (TypeError, ValueError):
        return default


def get_sweep_max():
    return _read_int(SWEEP_MAX_KEY, Config.EVAL_SWEEP_MAX, 1, 1000)


def get_sweep_trials():
    return _read_int(SWEEP_TRIALS_KEY, Config.EVAL_SWEEP_TRIALS, 1, 1000)


def set_sweep_params(sweep_max=None, sweep_trials=None):
    if sweep_max is not None:
        _set(SWEEP_MAX_KEY, max(1, min(1000, int(sweep_max))))
    if sweep_trials is not None:
        _set(SWEEP_TRIALS_KEY, max(1, min(1000, int(sweep_trials))))
    return get_sweep_max(), get_sweep_trials()


def sweep_trial_seed(trial):
    return Config.EVAL_SEED + SWEEP_SEED_OFFSET + trial * SWEEP_SEED_STRIDE


def get_sweep_trial_seeds():
    return [
        {"trial": trial + 1, "seed": sweep_trial_seed(trial)}
        for trial in range(get_sweep_trials())
    ]


def get_level_enabled(level_id):
    value = _get("level_enabled_%s" % level_id)
    if value is None:
        return True
    return value == "1"


def set_level_enabled(level_id, enabled):
    _set("level_enabled_%s" % level_id, "1" if enabled else "0")
    return get_level_enabled(level_id)


def get_levels():
    return [
        {"id": level["id"], "name": level["name"], "enabled": get_level_enabled(level["id"])}
        for level in LEVELS + PILOT_LEVELS
    ]
