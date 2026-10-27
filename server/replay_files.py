import os
import re
import shutil

from config import Config

REPLAY_ROOT = os.path.join(Config.UPLOAD_DIR, "replays")
MAX_CHUNK_BYTES = 8 * 1024 * 1024
MAX_FILE_BYTES = 256 * 1024 * 1024
_SAFE_NAME = re.compile(r"^[A-Za-z0-9_-]{1,80}$")


class ReplayFileError(ValueError):
    pass


def _check_name(name):
    if not _SAFE_NAME.match(name or ""):
        raise ReplayFileError("invalid replay name")
    return name


def relative_path(evaluation_id, upload_id, key):
    return "%s/%s/%s.zlib" % (_check_name(evaluation_id), _check_name(upload_id), _check_name(key))


def _absolute(relative):
    parts = (relative or "").split("/")
    if len(parts) != 3 or not parts[2].endswith(".zlib"):
        raise ReplayFileError("invalid replay path")
    _check_name(parts[0])
    _check_name(parts[1])
    _check_name(parts[2][: -len(".zlib")])
    return os.path.join(REPLAY_ROOT, *parts)


def write_chunk(evaluation_id, upload_id, key, offset, data):
    if len(data) > MAX_CHUNK_BYTES:
        raise ReplayFileError("chunk too large")
    relative = relative_path(evaluation_id, upload_id, key)
    path = _absolute(relative)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    size = os.path.getsize(path) if os.path.exists(path) else 0
    if offset > size:
        return relative, size
    if offset + len(data) > MAX_FILE_BYTES:
        raise ReplayFileError("replay too large")
    with open(path, "r+b" if size else "wb") as handle:
        handle.truncate(offset)
        handle.seek(offset)
        handle.write(data)
    return relative, offset + len(data)


def read(relative):
    try:
        with open(_absolute(relative), "rb") as handle:
            return handle.read()
    except (OSError, ReplayFileError):
        return None


def exists(relative):
    try:
        return os.path.isfile(_absolute(relative))
    except ReplayFileError:
        return False


def upload_ids(evaluation_id):
    folder = os.path.join(REPLAY_ROOT, _check_name(evaluation_id))
    if not os.path.isdir(folder):
        return []
    return [name for name in os.listdir(folder) if _SAFE_NAME.match(name)]


def remove_upload(evaluation_id, upload_id):
    folder = os.path.join(REPLAY_ROOT, _check_name(evaluation_id))
    shutil.rmtree(os.path.join(folder, _check_name(upload_id)), ignore_errors=True)
    try:
        os.rmdir(folder)
    except OSError:
        pass
