import secrets
from datetime import datetime, timedelta, timezone

from flask import request
from werkzeug.exceptions import Unauthorized

from config import Config
from database import db
from models import ADMIN_SESSION_TTL_SECONDS, AdminSession, AdminUser

# Sentinel returned when a request authenticates with the shared master key
# (break-glass) rather than a real admin user session.
MASTER_ADMIN = "master"


def _now():
    return datetime.now(timezone.utc)


def _aware(value):
    if value is not None and value.tzinfo is None:
        return value.replace(tzinfo=timezone.utc)
    return value


def extract_token():
    auth = request.headers.get("Authorization", "")
    if auth.startswith("Bearer "):
        return auth[len("Bearer "):].strip()
    return (request.headers.get("X-API-Key") or "").strip()


def create_session(user):
    token = secrets.token_urlsafe(32)
    session = AdminSession(
        token=token,
        user_id=user.id,
        created_at=_now(),
        expires_at=_now() + timedelta(seconds=ADMIN_SESSION_TTL_SECONDS),
        last_seen=_now(),
    )
    db.session.add(session)
    user.last_login = _now()
    db.session.commit()
    return session


def _session_user(token):
    if not token:
        return None
    session = db.session.get(AdminSession, token)
    if session is None:
        return None
    expires = _aware(session.expires_at)
    if expires is not None and expires < _now():
        db.session.delete(session)
        db.session.commit()
        return None
    session.last_seen = _now()
    db.session.commit()
    return db.session.get(AdminUser, session.user_id)


def revoke_token(token):
    if not token:
        return
    session = db.session.get(AdminSession, token)
    if session is not None:
        db.session.delete(session)
        db.session.commit()


def current_admin():
    # Returns the authenticated AdminUser, the MASTER_ADMIN sentinel, or None.
    token = extract_token()
    user = _session_user(token)
    if user is not None:
        return user
    if token and secrets.compare_digest(token, Config.API_SECRET_KEY):
        return MASTER_ADMIN
    return None


def require_admin():
    admin = current_admin()
    if admin is None:
        raise Unauthorized("Admin authentication required")
    return admin
