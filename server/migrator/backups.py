import json
import os
from datetime import datetime, timedelta, timezone

from config import Config

MANIFEST_NAME = "manifest.json"
MAX_BACKUP_AGE = timedelta(hours=24)


def backup_root():
    return Config.BACKUP_DIR


def read_manifest(folder):
    path = os.path.join(folder, MANIFEST_NAME)
    try:
        with open(path, encoding="utf-8") as handle:
            return json.load(handle)
    except (OSError, ValueError):
        return None


def list_backups():
    root = backup_root()
    if not os.path.isdir(root):
        return []
    backups = []
    for name in sorted(os.listdir(root)):
        folder = os.path.join(root, name)
        manifest = read_manifest(folder)
        if manifest is not None:
            backups.append((folder, manifest))
    return backups


def latest_verified_backup():
    verified = [
        (folder, manifest)
        for folder, manifest in list_backups()
        if manifest.get("verified") is True and manifest.get("created_at")
    ]
    if not verified:
        return None
    return max(verified, key=lambda item: item[1]["created_at"])


def recent_verified_backup():
    latest = latest_verified_backup()
    if latest is None:
        return None
    created_at = datetime.fromisoformat(latest[1]["created_at"])
    if datetime.now(timezone.utc) - created_at > MAX_BACKUP_AGE:
        return None
    return latest
