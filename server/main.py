import logging
import os
import time

from flask import Flask, abort, g, jsonify, request, send_from_directory
from flask_cors import CORS
from sqlalchemy import text

from config import Config
from database import db
from routers.evaluations import evaluations_bp
from routers.leaderboard import leaderboard_bp
from routers.runs import runs_bp
from routers.version import version_bp

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger(__name__)


def _ensure_columns():
    # Lightweight idempotent migrations for columns added after a table existed.
    statements = [
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS progress double precision DEFAULT 0",
    ]
    for statement in statements:
        try:
            db.session.execute(text(statement))
            db.session.commit()
        except Exception as exc:
            db.session.rollback()
            logger.warning("Migration skipped (%s): %s", statement, exc)


def _reset_stuck_evaluations():
    # Worker threads do not survive a restart, so in-flight runs would otherwise
    # stay "running" forever. Mark them failed so the UI stops polling zombies.
    try:
        db.session.execute(text(
            "UPDATE player_evaluations SET status='failed', "
            "error='interrupted by server restart' "
            "WHERE status IN ('queued', 'running')"
        ))
        db.session.commit()
    except Exception as exc:
        db.session.rollback()
        logger.warning("Could not reset stuck evaluations: %s", exc)


def create_app():
    app = Flask(__name__, static_folder=None)
    app.config.from_object(Config)

    CORS(app, origins=app.config["CORS_ORIGINS"])

    db.init_app(app)

    app.register_blueprint(runs_bp)
    app.register_blueprint(leaderboard_bp)
    app.register_blueprint(version_bp)
    app.register_blueprint(evaluations_bp)

    with app.app_context():
        db.create_all()
        _ensure_columns()
        _reset_stuck_evaluations()
        db.engine.dispose()

    @app.before_request
    def before_request():
        g.start_time = time.time()

    @app.after_request
    def after_request(response):
        if hasattr(g, "start_time"):
            if not request.path.startswith("/_app/"):
                elapsed = time.time() - g.start_time
                ms = round(elapsed * 1000, 2)
                logger.info(f"{request.method} {request.path} {response.status_code} - {ms}ms")
        return response

    client_dir = os.environ.get("CLIENT_DIR", "/app/client")

    @app.get("/health")
    def health():
        return jsonify({"status": "ok"})

    @app.route("/", defaults={"path": ""})
    @app.route("/<path:path>")
    def serve_client(path):
        if path.startswith("api/"):
            abort(404)

        if path:
            candidates = [path, f"{path}.html", os.path.join(path, "index.html")]
        else:
            candidates = ["index.html"]

        for candidate in candidates:
            if os.path.isfile(os.path.join(client_dir, candidate)):
                return send_from_directory(client_dir, candidate)

        # Unknown path. Only serve the SPA fallback for app routes. File/dotfile
        # probes (e.g. /.env, /.git/config, /env.py) that scanners hammer should
        # 404 instead of returning the SPA page.
        segments = [s for s in path.split("/") if s]
        is_dotpath = any(s.startswith(".") for s in segments)
        has_extension = "." in segments[-1] if segments else False
        if is_dotpath or has_extension:
            abort(404)

        if os.path.isfile(os.path.join(client_dir, "200.html")):
            return send_from_directory(client_dir, "200.html")

        abort(404)

    return app


app = create_app()

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5050)), debug=False)
