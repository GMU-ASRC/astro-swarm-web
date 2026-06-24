import logging
import os
import time

from flask import Flask, abort, g, jsonify, request, send_from_directory
from flask_cors import CORS

from config import Config
from database import db
from routers.evaluations import evaluations_bp
from routers.leaderboard import leaderboard_bp
from routers.runs import runs_bp
from routers.version import version_bp

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger(__name__)


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

        if os.path.isfile(os.path.join(client_dir, "200.html")):
            return send_from_directory(client_dir, "200.html")

        abort(404)

    return app


app = create_app()

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5050)), debug=False)
