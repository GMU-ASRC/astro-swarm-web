import logging
import os
import threading
from datetime import datetime, timedelta, timezone

from sqlalchemy import text

from database import db
from models import AppSetting

logger = logging.getLogger(__name__)

MEASURE_EVERY = timedelta(minutes=10)
MEASURE_LOCK_KEY = 734021118
BYTES_KEY = "uploads_bytes"
FILES_KEY = "uploads_files"
MEASURED_AT_KEY = "uploads_measured_at"

_measuring = threading.Lock()


def _setting(key):
    row = db.session.get(AppSetting, key)
    return row.value if row is not None else None


def _save_setting(key, value):
    row = db.session.get(AppSetting, key)
    if row is None:
        db.session.add(AppSetting(key=key, value=str(value)))
    else:
        row.value = str(value)


def cached_uploads_usage():
    measured_at = _setting(MEASURED_AT_KEY)
    return {
        "uploads_bytes": int(_setting(BYTES_KEY) or 0),
        "upload_files": int(_setting(FILES_KEY) or 0),
        "uploads_measured_at": measured_at,
    }


def _is_stale(measured_at):
    if not measured_at:
        return True
    try:
        measured = datetime.fromisoformat(measured_at)
    except ValueError:
        return True
    return datetime.now(timezone.utc) - measured > MEASURE_EVERY


def _walk_uploads(upload_dir):
    total_bytes = 0
    file_count = 0
    for folder, _, file_names in os.walk(upload_dir):
        for file_name in file_names:
            try:
                total_bytes += os.path.getsize(os.path.join(folder, file_name))
                file_count += 1
            except OSError:
                continue
    return total_bytes, file_count


def _measure(app):
    with app.app_context():
        try:
            got_lock = db.session.execute(
                text("SELECT pg_try_advisory_lock(:key)"), {"key": MEASURE_LOCK_KEY}
            ).scalar()
            if not got_lock:
                db.session.rollback()
                return
            try:
                total_bytes, file_count = _walk_uploads(app.config["UPLOAD_DIR"])
                _save_setting(BYTES_KEY, total_bytes)
                _save_setting(FILES_KEY, file_count)
                _save_setting(MEASURED_AT_KEY, datetime.now(timezone.utc).isoformat())
                db.session.commit()
            finally:
                db.session.execute(text("SELECT pg_advisory_unlock(:key)"), {"key": MEASURE_LOCK_KEY})
                db.session.commit()
        except Exception as error:
            db.session.rollback()
            logger.warning("Could not measure uploads: %s", error)
        finally:
            db.session.remove()
            _measuring.release()


def refresh_if_stale(app, force=False):
    if not force and not _is_stale(_setting(MEASURED_AT_KEY)):
        return False
    if not _measuring.acquire(blocking=False):
        return False
    threading.Thread(target=_measure, args=(app,), daemon=True).start()
    return True
