from datetime import datetime, timezone

from flask import Blueprint, Response, jsonify, request, stream_with_context
from werkzeug.exceptions import BadRequest

import backup_archive
from auth import require_admin

backup_bp = Blueprint("backup", __name__, url_prefix="/api/admin")

EXPORT_PARTS = ("database", "tables", "files")


def _requested_parts():
    raw = request.args.get("parts", "database,tables,files")
    parts = {part.strip() for part in raw.split(",") if part.strip()}
    unknown = parts - set(EXPORT_PARTS)
    if unknown:
        raise BadRequest("Unknown export parts: %s" % ", ".join(sorted(unknown)))
    if not parts:
        raise BadRequest("Choose at least one of: %s" % ", ".join(EXPORT_PARTS))
    return parts


@backup_bp.get("/export/info")
def export_info():
    require_admin()
    return jsonify({"pg_dump": backup_archive.pg_dump_available(), "parts": list(EXPORT_PARTS)})


@backup_bp.get("/export")
def export_data():
    require_admin()
    parts = _requested_parts()
    if "database" in parts and not backup_archive.pg_dump_available() and parts == {"database"}:
        raise BadRequest("pg_dump is not installed on the server, so a database dump cannot be made")

    stamp = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S")
    stream = backup_archive.stream_export("database" in parts, "tables" in parts, "files" in parts)
    response = Response(stream_with_context(stream), mimetype="application/zip")
    response.headers["Content-Disposition"] = 'attachment; filename="astroswarm-%s.zip"' % stamp
    response.headers["X-Accel-Buffering"] = "no"
    response.headers["Cache-Control"] = "no-store"
    return response
