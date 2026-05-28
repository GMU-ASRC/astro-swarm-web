import os
import uuid
from pathlib import Path

from flask import Blueprint, current_app, jsonify, request, send_file
from models import SimRun
from database import db
from godot_format import extract_metadata

runs_bp = Blueprint("runs", __name__, url_prefix="/api/runs")

RUNS_DIR = "runs"


def run_upload_path(stored_filename: str, upload_dir: str) -> Path:
    return Path(upload_dir) / RUNS_DIR / stored_filename


@runs_bp.get("")
def list_runs():
    page = request.args.get("page", 1, type=int)
    page_size = request.args.get("page_size", 20, type=int)

    pagination = (
        SimRun.query.order_by(SimRun.created_at.desc())
        .paginate(page=page, per_page=page_size, error_out=False)
    )

    return jsonify({
        "items": [item.to_list_dict() for item in pagination.items],
        "total": pagination.total,
        "page": page,
        "page_size": page_size,
    })


@runs_bp.post("")
def upload_run():
    if "file" not in request.files:
        return jsonify({"error": "No file provided."}), 400

    file = request.files["file"]
    title = request.form.get("title", "").strip()
    description = request.form.get("description", "").strip()
    author = request.form.get("author", "anonymous").strip() or "anonymous"

    if not title:
        return jsonify({"error": "Title is required."}), 400

    if not file.filename or Path(file.filename).suffix.lower() != ".run":
        return jsonify({"error": "Only .run files are accepted."}), 400

    content = file.read()
    if len(content) > current_app.config["MAX_UPLOAD_BYTES"]:
        return jsonify({"error": "File exceeds maximum size of 10 MB."}), 413

    stored_filename = f"{uuid.uuid4()}.run"
    upload_dir = current_app.config["UPLOAD_DIR"]
    dest = run_upload_path(stored_filename, upload_dir)
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_bytes(content)

    try:
        metadata = extract_metadata(content)
    except Exception:
        metadata = {}

    run = SimRun(
        title=title[:80],
        description=description[:400],
        author=author[:60],
        original_filename=file.filename,
        stored_filename=stored_filename,
        file_size=len(content),
        species=metadata.get("species", []),
        robot_count=metadata.get("robot_count", 0),
        frame_count=metadata.get("frame_count", 0),
        arena_width=metadata.get("arena_width", 1280.0),
        arena_height=metadata.get("arena_height", 720.0),
    )

    db.session.add(run)
    db.session.commit()

    return jsonify(run.to_dict()), 201


@runs_bp.get("/<run_id>")
def get_run(run_id: str):
    run = SimRun.query.get_or_404(run_id, description="Run not found.")
    return jsonify(run.to_dict())


@runs_bp.get("/<run_id>/download")
def download_run(run_id: str):
    run = SimRun.query.get_or_404(run_id, description="Run not found.")

    file_path = run_upload_path(run.stored_filename, current_app.config["UPLOAD_DIR"])
    if not file_path.exists():
        return jsonify({"error": "File not found on server."}), 404

    run.download_count += 1
    db.session.commit()

    return send_file(str(file_path), download_name=run.original_filename, as_attachment=True)


@runs_bp.delete("/<run_id>")
def delete_run(run_id: str):
    run = SimRun.query.get_or_404(run_id, description="Run not found.")

    file_path = run_upload_path(run.stored_filename, current_app.config["UPLOAD_DIR"])
    if file_path.exists():
        os.remove(file_path)

    db.session.delete(run)
    db.session.commit()

    return "", 204
