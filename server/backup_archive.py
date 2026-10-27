import json
import os
import shutil
import subprocess
import zipfile
from datetime import date, datetime, timezone
from decimal import Decimal

from config import Config
from database import db

READ_BLOCK_BYTES = 1024 * 1024
PRECOMPRESSED_SUFFIXES = (".zlib", ".zip", ".gz", ".png", ".jpg", ".jpeg", ".webp", ".mp4", ".webm", ".dump")
TABLE_BATCH_ROWS = 500


class _ChunkSink:
    def __init__(self):
        self._chunks = []

    def write(self, data):
        self._chunks.append(bytes(data))
        return len(data)

    def flush(self):
        pass

    def drain(self):
        data = b"".join(self._chunks)
        self._chunks.clear()
        return data


def _json_default(value):
    if isinstance(value, (datetime, date)):
        return value.isoformat()
    if isinstance(value, Decimal):
        return float(value)
    if isinstance(value, (bytes, bytearray, memoryview)):
        return bytes(value).hex()
    return str(value)


def _entry(name, compress=True):
    info = zipfile.ZipInfo(name, date_time=datetime.now().timetuple()[:6])
    info.compress_type = zipfile.ZIP_DEFLATED if compress else zipfile.ZIP_STORED
    return info


def pg_dump_available():
    return shutil.which("pg_dump") is not None


def _database_url():
    url = Config.SQLALCHEMY_DATABASE_URI
    for driver in ("postgresql+psycopg2://", "postgresql+psycopg://"):
        if url.startswith(driver):
            return "postgresql://" + url[len(driver):]
    return url


def _stream_pg_dump(archive, sink):
    process = subprocess.Popen(
        ["pg_dump", "--format=custom", "--no-owner", "--no-privileges", "--dbname", _database_url()],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    with archive.open(_entry("database/database.dump", compress=False), "w", force_zip64=True) as target:
        for block in iter(lambda: process.stdout.read(READ_BLOCK_BYTES), b""):
            target.write(block)
            yield sink.drain()
    error = process.stderr.read().decode("utf-8", "replace")
    if process.wait() != 0:
        archive.writestr(_entry("database/pg_dump_error.txt"), error)
    yield sink.drain()


def _stream_tables(archive, sink):
    counts = {}
    for table in db.metadata.sorted_tables:
        rows = 0
        with archive.open(_entry("tables/%s.jsonl" % table.name), "w", force_zip64=True) as target:
            result = db.session.execute(table.select().execution_options(yield_per=TABLE_BATCH_ROWS))
            for row in result.mappings():
                line = json.dumps(dict(row), default=_json_default, separators=(",", ":")) + "\n"
                target.write(line.encode("utf-8"))
                rows += 1
                if rows % TABLE_BATCH_ROWS == 0:
                    yield sink.drain()
        counts[table.name] = rows
        yield sink.drain()
    db.session.rollback()
    archive.writestr(_entry("tables/row_counts.json"), json.dumps(counts, indent=2))
    yield sink.drain()


def _stream_files(archive, sink):
    root = Config.UPLOAD_DIR
    if not os.path.isdir(root):
        return
    for folder, _, names in os.walk(root):
        for name in sorted(names):
            path = os.path.join(folder, name)
            relative = os.path.relpath(path, root).replace(os.sep, "/")
            compress = not name.lower().endswith(PRECOMPRESSED_SUFFIXES)
            try:
                source = open(path, "rb")
            except OSError:
                continue
            with source, archive.open(_entry("uploads/" + relative, compress), "w", force_zip64=True) as target:
                for block in iter(lambda: source.read(READ_BLOCK_BYTES), b""):
                    target.write(block)
                    yield sink.drain()


def stream_export(include_database, include_tables, include_files):
    sink = _ChunkSink()
    with zipfile.ZipFile(sink, "w", allowZip64=True) as archive:
        manifest = {
            "created_at": datetime.now(timezone.utc).isoformat(),
            "database_dump": include_database and pg_dump_available(),
            "tables": include_tables,
            "files": include_files,
            "restore": "pg_restore --clean --if-exists --no-owner --dbname <database url> database/database.dump",
        }
        archive.writestr(_entry("manifest.json"), json.dumps(manifest, indent=2))
        yield sink.drain()
        if manifest["database_dump"]:
            yield from _stream_pg_dump(archive, sink)
        if include_tables:
            yield from _stream_tables(archive, sink)
        if include_files:
            yield from _stream_files(archive, sink)
    yield sink.drain()
