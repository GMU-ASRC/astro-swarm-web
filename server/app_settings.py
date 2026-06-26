from config import Config
from database import db
from models import AppSetting

ENEMY_X_KEY = "enemy_start_x"
ENEMY_Y_KEY = "enemy_start_y"


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
