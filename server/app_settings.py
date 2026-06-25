from config import Config
from database import db
from models import AppSetting

MAX_JOBS_KEY = "max_jobs"
JOBS_HARD_CAP = 20


def get_max_jobs():
    setting = db.session.get(AppSetting, MAX_JOBS_KEY)
    if setting is None or setting.value is None:
        return Config.EVAL_MAX_JOBS
    try:
        value = int(setting.value)
    except (TypeError, ValueError):
        return Config.EVAL_MAX_JOBS
    return max(1, min(JOBS_HARD_CAP, value))


def set_max_jobs(value):
    value = max(1, min(JOBS_HARD_CAP, int(value)))
    setting = db.session.get(AppSetting, MAX_JOBS_KEY)
    if setting is None:
        setting = AppSetting(key=MAX_JOBS_KEY, value=str(value))
        db.session.add(setting)
    else:
        setting.value = str(value)
    db.session.commit()
    return value
