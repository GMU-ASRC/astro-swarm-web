import os

from flask import Flask, abort, jsonify, send_from_directory
from flask_cors import CORS

from config import Config
from database import db
from routers.configs import configs_bp
from routers.runs import runs_bp
from routers.leaderboard import leaderboard_bp


def create_app():
    app = Flask(__name__, static_folder=None)
    app.config.from_object(Config)

    CORS(app, origins=app.config["CORS_ORIGINS"])

    db.init_app(app)

    app.register_blueprint(configs_bp)
    app.register_blueprint(runs_bp)
    app.register_blueprint(leaderboard_bp)

    with app.app_context():
        db.create_all()
        db.engine.dispose()

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
