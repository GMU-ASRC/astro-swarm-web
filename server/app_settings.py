from config import Config
from database import db
from models import AppSetting

MAX_JOBS_KEY = "max_jobs"
ENEMY_X_KEY = "enemy_start_x"
ENEMY_Y_KEY = "enemy_start_y"
JOBS_HARD_CAP = 20


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


def get_max_jobs():
    value = _get(MAX_JOBS_KEY)
    if value is None:
        return Config.EVAL_MAX_JOBS
    try:
        return max(1, min(JOBS_HARD_CAP, int(value)))
    except (TypeError, ValueError):
        return Config.EVAL_MAX_JOBS


def set_max_jobs(value):
    value = max(1, min(JOBS_HARD_CAP, int(value)))
    _set(MAX_JOBS_KEY, value)
    return value


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
