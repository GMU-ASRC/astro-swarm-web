import hashlib
import json

import job_settings
from app_settings import is_pilot_level
from config import Config


def content_hash(evaluation, job_config):
    content = {
        "level_id": evaluation.level_id or "farp",
        "algorithm": evaluation.algorithm or [],
        "placements": evaluation.placements or [],
        "trials": evaluation.trials,
        "config": job_config,
        "simulator_version": Config.SIMULATOR_VERSION,
    }
    schedule = evaluation.genome_schedule or []
    if len(schedule) > 1:
        content["genome_schedule"] = schedule
    encoded = json.dumps(content, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()


def for_new_submission(evaluation):
    if is_pilot_level(evaluation.level_id):
        return None
    return content_hash(evaluation, job_settings.config_for(evaluation))


def for_finished_job(evaluation):
    if is_pilot_level(evaluation.level_id):
        return None
    job_config = evaluation.job_config or job_settings.config_for(evaluation)
    return content_hash(evaluation, job_config)
