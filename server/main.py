import html
import logging
import os
import time

from flask import Flask, Response, abort, g, jsonify, request, send_from_directory
from flask_cors import CORS
from sqlalchemy import text
from werkzeug.exceptions import HTTPException

from config import Config
from database import db
from models import PlayerEvaluation
from routers.admin import admin_bp
from routers.evaluations import evaluations_bp
from routers.leaderboard import leaderboard_bp
from routers.runs import runs_bp
from routers.version import version_bp
from routers.workers import workers_bp

SITE_DESCRIPTION = (
    "Design species, program behavior, unleash your swarm. AstroSwarm — a pixel-art "
    "tower-defense and swarm simulator built in Godot 4."
)

PAGE_META = {
    "": ("AstroSwarm", SITE_DESCRIPTION),
    "previews": ("Previews — AstroSwarm", "Screenshots and preview footage from AstroSwarm."),
    "downloads": ("Downloads — AstroSwarm", "Download the latest AstroSwarm builds."),
    "leaderboard": ("XP Leaderboard — AstroSwarm", "Top commanders ranked by XP earned across every AstroSwarm level."),
    "levels": ("Levels — AstroSwarm", "Per-level benchmark data for player algorithms in AstroSwarm."),
    "simulator": ("Simulator — AstroSwarm", "Shared swarm-behaviour simulations from the AstroSwarm sandbox."),
}


def _base_url():
    if Config.PUBLIC_BASE_URL:
        return Config.PUBLIC_BASE_URL.rstrip("/")
    return request.url_root.rstrip("/")


def _resolve_meta(path):
    base = _base_url()
    segments = [s for s in path.split("/") if s]
    title, description = PAGE_META.get(segments[0] if segments else "", ("AstroSwarm", SITE_DESCRIPTION))
    meta_type = "website"
    image = f"{base}/previews/Pasted%20image.png"

    if len(segments) >= 2 and segments[0] == "levels":
        evaluation = db.session.get(PlayerEvaluation, segments[1])
        if evaluation is not None:
            results = evaluation.results if isinstance(evaluation.results, dict) else {}
            rate = results.get("success_rate")
            title = f"{evaluation.username} — FARP Benchmark — AstroSwarm"
            if rate is not None:
                description = f"{rate}% detection rate over {evaluation.trials} trials in the AstroSwarm FARP defender benchmark."
            else:
                description = f"FARP defender benchmark for {evaluation.username} in AstroSwarm."
            meta_type = "article"
            image = f"{base}/api/evaluations/{evaluation.id}/thumbnail.png"

    return {"title": title, "description": description, "type": meta_type, "image": image, "url": f"{base}/{path}"}


def _render_html(client_dir, filename, path):
    with open(os.path.join(client_dir, filename), encoding="utf-8") as f:
        page = f.read()
    try:
        meta = _resolve_meta(path)
    except Exception:
        db.session.rollback()
        base = _base_url()
        meta = {"title": "AstroSwarm", "description": SITE_DESCRIPTION, "type": "website", "image": f"{base}/previews/Pasted%20image.png", "url": f"{base}/{path}"}
    tokens = {
        "__META_TITLE__": meta["title"],
        "__META_DESC__": meta["description"],
        "__META_TYPE__": meta["type"],
        "__META_IMAGE__": meta["image"],
        "__META_URL__": meta["url"],
    }
    for token, value in tokens.items():
        page = page.replace(token, html.escape(value, quote=True))
    return Response(page, mimetype="text/html")

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger(__name__)


def _ensure_columns():
    # Lightweight idempotent migrations for columns added after a table existed.
    statements = [
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS progress double precision DEFAULT 0",
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS replays json DEFAULT '[]'::json",
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS placements json DEFAULT '[]'::json",
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS level_id varchar(40) DEFAULT 'farp'",
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS worker_id varchar(64)",
        "ALTER TABLE sim_runs ADD COLUMN IF NOT EXISTS thumbnail_filename varchar(255)",
        "ALTER TABLE sim_runs ADD COLUMN IF NOT EXISTS video_filename varchar(255)",
        "ALTER TABLE sim_runs ADD COLUMN IF NOT EXISTS frame_count integer DEFAULT 0",
        "ALTER TABLE workers ADD COLUMN IF NOT EXISTS system_stats json",
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS stage varchar(200)",
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS game_version varchar(20) DEFAULT 'v0.0.4'",
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS defender_count integer DEFAULT 0",
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS xp_awarded integer",
        "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS collisions boolean DEFAULT false",
    ]
    for statement in statements:
        try:
            db.session.execute(text(statement))
            db.session.commit()
        except Exception as exc:
            db.session.rollback()
            logger.warning("Migration skipped (%s): %s", statement, exc)


def _ensure_admin_user():
    # Seed an initial admin login if none exist. Defaults to username "admin"
    # with the shared API key as the password, both overridable via env. The
    # operator should sign in and change the password / add users.
    try:
        from models import AdminUser

        if AdminUser.query.count() > 0:
            return
        username = os.environ.get("ADMIN_USERNAME", "admin")
        password = os.environ.get("ADMIN_PASSWORD") or Config.API_SECRET_KEY
        user = AdminUser(username=username)
        user.set_password(password)
        db.session.add(user)
        db.session.commit()
        logger.info("Created initial admin user '%s'", username)
    except Exception as exc:
        db.session.rollback()
        logger.warning("Could not create initial admin user: %s", exc)


def _recover_shards_on_restart():
    # Jobs are split into shards that run on external worker nodes. On a server
    # restart, return any in-flight shards to the queue (completed shards keep
    # their stored result), and rebuild shards for any in-flight evaluation that
    # has none (e.g. jobs queued before this version).
    try:
        from models import EvaluationShard, PlayerEvaluation
        from routers.workers import create_shards

        db.session.execute(text(
            "UPDATE evaluation_shards SET status='queued', worker_id=NULL, done_units=0 "
            "WHERE status='running'"
        ))
        db.session.commit()

        pending = PlayerEvaluation.query.filter(
            PlayerEvaluation.status.in_(("queued", "running"))
        ).all()
        for evaluation in pending:
            has_shards = EvaluationShard.query.filter_by(evaluation_id=evaluation.id).count()
            if not has_shards:
                evaluation.status = "queued"
                evaluation.progress = 0.0
                evaluation.worker_id = None
                create_shards(evaluation)
        db.session.commit()
    except Exception as exc:
        db.session.rollback()
        logger.warning("Could not recover evaluation shards: %s", exc)


def create_app():
    app = Flask(__name__, static_folder=None)
    app.config.from_object(Config)

    CORS(app, origins=app.config["CORS_ORIGINS"])

    db.init_app(app)

    app.register_blueprint(runs_bp)
    app.register_blueprint(leaderboard_bp)
    app.register_blueprint(version_bp)
    app.register_blueprint(evaluations_bp)
    app.register_blueprint(workers_bp)
    app.register_blueprint(admin_bp)

    @app.errorhandler(HTTPException)
    def json_error(exc):
        # API clients (the game included) parse every response as JSON, so keep
        # errors machine-readable instead of Werkzeug's default HTML page.
        return jsonify({"error": exc.description, "status": exc.code}), exc.code

    with app.app_context():
        db.create_all()
        _ensure_columns()
        _ensure_admin_user()
        _recover_shards_on_restart()
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
                if candidate.endswith(".html"):
                    return _render_html(client_dir, candidate, path)
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
            return _render_html(client_dir, "200.html", path)

        abort(404)

    return app


app = create_app()

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5050)), debug=False)
