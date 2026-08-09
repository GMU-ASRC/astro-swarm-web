import os

from flask import Blueprint, current_app, jsonify, request
from sqlalchemy import text
from werkzeug.exceptions import BadRequest, Conflict, Forbidden, NotFound

from auth import (
    MASTER_ADMIN,
    create_session,
    current_admin,
    extract_token,
    require_admin,
    revoke_token,
)
from database import db
from models import AdminSession, AdminUser

admin_bp = Blueprint("admin", __name__, url_prefix="/api/admin")

TRACKED_TABLES = [
    "player_evaluations",
    "evaluation_shards",
    "sim_runs",
    "sim_configs",
    "survive_matches",
    "leaderboard_entries",
    "workers",
    "admin_users",
    "admin_sessions",
    "app_settings",
]


def _username_field(data):
    return str(data.get("username", "")).strip()


@admin_bp.post("/login")
def login():
    data = request.get_json(silent=True) or {}
    username = _username_field(data)
    password = str(data.get("password", ""))
    if not username or not password:
        raise BadRequest("username and password are required")

    user = AdminUser.query.filter_by(username=username).first()
    if user is None or not user.check_password(password):
        return jsonify({"error": "Invalid username or password"}), 401

    session = create_session(user)
    return jsonify({
        "token": session.token,
        "expires_at": session.expires_at.isoformat() if session.expires_at else None,
        "user": user.to_dict(),
    })


@admin_bp.post("/logout")
def logout():
    revoke_token(extract_token())
    return jsonify({"ok": True})


@admin_bp.get("/me")
def me():
    admin = current_admin()
    if admin is None:
        return jsonify({"error": "Not authenticated"}), 401
    if admin is MASTER_ADMIN:
        return jsonify({"username": "master", "master": True})
    return jsonify({**admin.to_dict(), "master": False})


@admin_bp.post("/password")
def change_password():
    admin = require_admin()
    data = request.get_json(silent=True) or {}
    new_password = str(data.get("new_password", ""))
    if len(new_password) < 8:
        raise BadRequest("new_password must be at least 8 characters")
    if admin is MASTER_ADMIN:
        raise Forbidden("The master key login cannot change a user password")
    current_password = str(data.get("current_password", ""))
    if not admin.check_password(current_password):
        return jsonify({"error": "Current password is incorrect"}), 401
    admin.set_password(new_password)
    db.session.commit()
    return jsonify({"ok": True})


@admin_bp.get("/users")
def list_users():
    require_admin()
    users = AdminUser.query.order_by(AdminUser.username.asc()).all()
    return jsonify([user.to_dict() for user in users])


@admin_bp.post("/users")
def create_user():
    require_admin()
    data = request.get_json(silent=True) or {}
    username = _username_field(data)
    password = str(data.get("password", ""))
    if not username:
        raise BadRequest("username is required")
    if len(password) < 8:
        raise BadRequest("password must be at least 8 characters")
    if AdminUser.query.filter_by(username=username).first() is not None:
        raise Conflict("That username already exists")
    user = AdminUser(username=username)
    user.set_password(password)
    db.session.add(user)
    db.session.commit()
    return jsonify(user.to_dict()), 201


def _database_size():
    # pg_total_relation_size covers the table plus its indexes and TOAST data,
    # which is where the compressed replay blobs actually live.
    total = db.session.execute(
        text("SELECT pg_database_size(current_database())")
    ).scalar()
    tables = []
    for name in TRACKED_TABLES:
        try:
            size = db.session.execute(
                text("SELECT pg_total_relation_size(:name)"), {"name": name}
            ).scalar()
            rows = db.session.execute(text(f"SELECT count(*) FROM {name}")).scalar()
        except Exception:
            db.session.rollback()
            continue
        tables.append({"name": name, "bytes": int(size or 0), "rows": int(rows or 0)})
    tables.sort(key=lambda table: table["bytes"], reverse=True)
    return int(total or 0), tables


def _uploads_size(upload_dir):
    total = 0
    files = 0
    for root, _dirs, names in os.walk(upload_dir):
        for name in names:
            try:
                total += os.path.getsize(os.path.join(root, name))
                files += 1
            except OSError:
                continue
    return total, files


@admin_bp.get("/storage")
def storage():
    require_admin()

    try:
        database_bytes, tables = _database_size()
        database_error = None
    except Exception as exc:
        db.session.rollback()
        database_bytes, tables = 0, []
        database_error = str(exc)

    upload_dir = current_app.config["UPLOAD_DIR"]
    if os.path.isdir(upload_dir):
        uploads_bytes, upload_files = _uploads_size(upload_dir)
        uploads_error = None
    else:
        uploads_bytes, upload_files = 0, 0
        uploads_error = "Upload directory not found"

    return jsonify({
        "total_bytes": database_bytes + uploads_bytes,
        "database_bytes": database_bytes,
        "database_error": database_error,
        "tables": tables,
        "uploads_bytes": uploads_bytes,
        "upload_files": upload_files,
        "upload_dir": upload_dir,
        "uploads_error": uploads_error,
    })


@admin_bp.delete("/users/<user_id>")
def delete_user(user_id):
    admin = require_admin()
    target = db.session.get(AdminUser, user_id)
    if target is None:
        raise NotFound("User not found")
    if AdminUser.query.count() <= 1:
        raise BadRequest("Cannot delete the last admin user")
    if admin is not MASTER_ADMIN and admin.id == target.id:
        raise BadRequest("You cannot delete the account you are signed in with")
    AdminSession.query.filter_by(user_id=target.id).delete(synchronize_session=False)
    db.session.delete(target)
    db.session.commit()
    return jsonify({"ok": True})
