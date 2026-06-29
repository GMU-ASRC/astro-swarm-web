import io
import json
import os
import re
import subprocess
import tempfile
import uuid
import zipfile
from pathlib import Path

from flask import Blueprint, current_app, jsonify, request, send_file
from auth import require_admin
from models import SimRun
from database import db
from godot_format import extract_metadata

runs_bp = Blueprint("runs", __name__, url_prefix="/api/runs")


def safe_filename(value: str) -> str:
    cleaned = re.sub(r"[^A-Za-z0-9_-]+", "_", value or "").strip("_")
    return cleaned or "run"

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
    if "run_file" not in request.files or "config_file" not in request.files or "video_file" not in request.files:
        return jsonify({"error": "Missing run_file, config_file, or video_file."}), 400

    run_file = request.files["run_file"]
    config_file = request.files["config_file"]
    video_file = request.files["video_file"]
    
    title = request.form.get("title", "").strip()
    description = request.form.get("description", "").strip()
    author = request.form.get("author", "anonymous").strip() or "anonymous"

    if not title:
        return jsonify({"error": "Title is required."}), 400

    if not run_file.filename or Path(run_file.filename).suffix.lower() != ".run":
        return jsonify({"error": "run_file must be a .run file."}), 400

    run_content = run_file.read()
    config_content = config_file.read()
    video_content = video_file.read()
    
    total_size = len(run_content) + len(config_content) + len(video_content)
    if total_size > current_app.config.get("MAX_UPLOAD_BYTES", 10 * 1024 * 1024):
        return jsonify({"error": "Total file size exceeds maximum size limit."}), 413

    upload_dir = current_app.config["UPLOAD_DIR"]
    run_uuid = str(uuid.uuid4())
    stored_filename = f"{run_uuid}.zip"
    thumbnail_filename = f"{run_uuid}.jpg"
    
    dest = run_upload_path(stored_filename, upload_dir)
    dest.parent.mkdir(parents=True, exist_ok=True)
    
    # Compress files into zip
    with zipfile.ZipFile(dest, "w", compression=zipfile.ZIP_DEFLATED) as zf:
        zf.writestr(run_file.filename or "run.run", run_content)
        zf.writestr(config_file.filename or "config.json", config_content)
        zf.writestr(video_file.filename or "video.mp4", video_content)
        
    file_size = dest.stat().st_size
    
    # Save video separately for streaming
    video_filename = f"{run_uuid}.mp4"
    video_dest = run_upload_path(video_filename, upload_dir)
    with open(video_dest, "wb") as f:
        f.write(video_content)
        
    try:
        subprocess.run(
            ["ffmpeg", "-y", "-i", str(video_dest), "-vframes", "1", "-f", "image2", str(thumbnail_dest)],
            check=True,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL
        )
    except Exception as e:
        # Ignore thumbnail extraction failure if it happens
        thumbnail_filename = None

    try:
        metadata = extract_metadata(run_content)
    except Exception:
        metadata = {}

    run = SimRun(
        title=title[:80],
        description=description[:400],
        author=author[:60],
        original_filename=f"{title}.zip",
        stored_filename=stored_filename,
        thumbnail_filename=thumbnail_filename,
        video_filename=video_filename,
        file_size=file_size,
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


@runs_bp.get("/<run_id>/export")
def export_run(run_id: str):
    run = SimRun.query.get_or_404(run_id, description="Run not found.")

    config_data = None
    zip_path = run_upload_path(run.stored_filename, current_app.config["UPLOAD_DIR"])
    if zip_path.exists():
        try:
            with zipfile.ZipFile(zip_path, "r") as source:
                config_files = [f for f in source.namelist() if f.endswith(".cfg") or f.endswith(".json")]
                if config_files:
                    config_data = source.read(config_files[0])
        except Exception:
            config_data = None

    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("run.json", json.dumps(run.to_dict(), indent=2))
        if config_data is not None:
            archive.writestr("config.json", config_data)
    buffer.seek(0)

    download_name = f"run_{safe_filename(run.title)}_{run.id}.zip"
    return send_file(
        buffer,
        mimetype="application/zip",
        as_attachment=True,
        download_name=download_name,
    )


@runs_bp.get("/<run_id>/download")
def download_run(run_id: str):
    run = SimRun.query.get_or_404(run_id, description="Run not found.")

    file_path = run_upload_path(run.stored_filename, current_app.config["UPLOAD_DIR"])
    if not file_path.exists():
        return jsonify({"error": "File not found on server."}), 404

    run.download_count += 1
    db.session.commit()

    return send_file(str(file_path), download_name=run.original_filename, as_attachment=True)

@runs_bp.get("/<run_id>/thumbnail")
def get_thumbnail(run_id: str):
    run = SimRun.query.get_or_404(run_id, description="Run not found.")
    if not run.thumbnail_filename:
        return jsonify({"error": "No thumbnail available."}), 404
        
    file_path = run_upload_path(run.thumbnail_filename, current_app.config["UPLOAD_DIR"])
    if not file_path.exists():
        return jsonify({"error": "Thumbnail file not found on server."}), 404
        
    return send_file(str(file_path), mimetype="image/jpeg")


@runs_bp.get("/<run_id>/video")
def get_video(run_id: str):
    run = SimRun.query.get_or_404(run_id, description="Run not found.")
    
    if run.video_filename:
        file_path = run_upload_path(run.video_filename, current_app.config["UPLOAD_DIR"])
        if file_path.exists():
            return send_file(str(file_path), mimetype="video/mp4")
            
    # Fallback to extracting from zip for older runs without video_filename
    zip_path = run_upload_path(run.stored_filename, current_app.config["UPLOAD_DIR"])
    if not zip_path.exists():
        return jsonify({"error": "Video file not found on server."}), 404
        
    # Extract to a temp file and serve
    import tempfile
    with zipfile.ZipFile(zip_path, "r") as zf:
        video_files = [f for f in zf.namelist() if f.endswith(".mp4") or f.endswith(".webm")]
        if not video_files:
            return jsonify({"error": "No video found in run archive."}), 404
            
        video_data = zf.read(video_files[0])
        
    fd, tmp_path = tempfile.mkstemp(suffix=".mp4")
    with os.fdopen(fd, "wb") as f:
        f.write(video_data)
        
    return send_file(tmp_path, mimetype="video/mp4")


@runs_bp.get("/<run_id>/config")
def get_config(run_id: str):
    run = SimRun.query.get_or_404(run_id, description="Run not found.")
    
    zip_path = run_upload_path(run.stored_filename, current_app.config["UPLOAD_DIR"])
    if not zip_path.exists():
        return jsonify({"error": "Run archive not found on server."}), 404
        
    try:
        with zipfile.ZipFile(zip_path, "r") as zf:
            config_files = [f for f in zf.namelist() if f.endswith(".cfg") or f.endswith(".json")]
            if not config_files:
                return jsonify({"error": "No config file found in run archive."}), 404
                
            config_data = zf.read(config_files[0])
            
        import json
        return jsonify(json.loads(config_data))
    except Exception as e:
        return jsonify({"error": f"Failed to parse config: {str(e)}"}), 500


@runs_bp.delete("/<run_id>")
def delete_run(run_id: str):
    require_admin()
    run = SimRun.query.get_or_404(run_id, description="Run not found.")

    upload_dir = current_app.config["UPLOAD_DIR"]
    file_path = run_upload_path(run.stored_filename, upload_dir)
    if file_path.exists():
        os.remove(file_path)
        
        
    if run.thumbnail_filename:
        thumb_path = run_upload_path(run.thumbnail_filename, upload_dir)
        if thumb_path.exists():
            os.remove(thumb_path)
            
    if run.video_filename:
        video_path = run_upload_path(run.video_filename, upload_dir)
        if video_path.exists():
            os.remove(video_path)

    db.session.delete(run)
    db.session.commit()

    return "", 204
