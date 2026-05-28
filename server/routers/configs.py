import os
import uuid
from pathlib import Path

from flask import Blueprint, current_app, jsonify, request, send_file
from models import SimConfig
from database import db
from godot_format import extract_metadata

configs_bp = Blueprint("configs", __name__, url_prefix="/api/configs")


def allowed_file(filename: str) -> bool:
    ext = Path(filename).suffix.lower()
    return ext in current_app.config["ALLOWED_EXTENSIONS"]


@configs_bp.get("")
def list_configs():
    page = request.args.get("page", 1, type=int)
    page_size = request.args.get("page_size", 20, type=int)

    pagination = (
        SimConfig.query.order_by(SimConfig.created_at.desc())
        .paginate(page=page, per_page=page_size, error_out=False)
    )

    return jsonify({
        "items": [item.to_list_dict() for item in pagination.items],
        "total": pagination.total,
        "page": page,
        "page_size": page_size,
    })


@configs_bp.post("")
def upload_config():
    if "file" not in request.files:
        return jsonify({"error": "No file provided."}), 400

    file = request.files["file"]
    title = request.form.get("title", "").strip()
    description = request.form.get("description", "").strip()
    author = request.form.get("author", "anonymous").strip() or "anonymous"

    if not title:
        return jsonify({"error": "Title is required."}), 400

    if not file.filename or not allowed_file(file.filename):
        return jsonify({"error": "File type not allowed. Must be .cfg or .run"}), 400

    content = file.read()
    if len(content) > current_app.config["MAX_UPLOAD_BYTES"]:
        return jsonify({"error": "File exceeds maximum size of 10 MB."}), 413

    extension = Path(file.filename).suffix.lower()
    stored_filename = f"{uuid.uuid4()}{extension}"
    upload_dir = Path(current_app.config["UPLOAD_DIR"])
    upload_dir.mkdir(parents=True, exist_ok=True)

    (upload_dir / stored_filename).write_bytes(content)

    try:
        metadata = extract_metadata(content)
    except Exception:
        metadata = {}

    config = SimConfig(
        title=title[:80],
        description=description[:400],
        author=author[:60],
        original_filename=file.filename,
        stored_filename=stored_filename,
        file_type=extension.lstrip("."),
        file_size=len(content),
        species=metadata.get("species", []),
        robot_count=metadata.get("robot_count", 0),
        arena_width=metadata.get("arena_width", 1280.0),
        arena_height=metadata.get("arena_height", 720.0),
    )

    db.session.add(config)
    db.session.commit()

    return jsonify(config.to_dict()), 201


@configs_bp.get("/<config_id>")
def get_config(config_id: str):
    config = SimConfig.query.get_or_404(config_id, description="Config not found.")
    return jsonify(config.to_dict())


@configs_bp.get("/<config_id>/download")
def download_config(config_id: str):
    config = SimConfig.query.get_or_404(config_id, description="Config not found.")

    file_path = Path(current_app.config["UPLOAD_DIR"]) / config.stored_filename
    if not file_path.exists():
        return jsonify({"error": "File not found on server."}), 404

    config.download_count += 1
    db.session.commit()

    return send_file(str(file_path), download_name=config.original_filename, as_attachment=True)


@configs_bp.delete("/<config_id>")
def delete_config(config_id: str):
    config = SimConfig.query.get_or_404(config_id, description="Config not found.")

    file_path = Path(current_app.config["UPLOAD_DIR"]) / config.stored_filename
    if file_path.exists():
        os.remove(file_path)

    db.session.delete(config)
    db.session.commit()

    return "", 204
