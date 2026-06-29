from flask import Blueprint, jsonify, request
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
