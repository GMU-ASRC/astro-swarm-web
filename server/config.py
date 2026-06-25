import os
from dotenv import load_dotenv

load_dotenv()


class Config:
    SQLALCHEMY_DATABASE_URI = os.environ["DATABASE_URL"]
    SQLALCHEMY_TRACK_MODIFICATIONS = False
    UPLOAD_DIR = os.environ.get("UPLOAD_DIR", "/app/uploads")
    MAX_UPLOAD_BYTES = 10 * 1024 * 1024
    ALLOWED_EXTENSIONS = {".cfg", ".run"}
    CORS_ORIGINS = os.environ.get("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000").split(",")
    API_SECRET_KEY = os.environ.get("API_SECRET_KEY", "dev_secret_key")

    EVAL_SEED = int(os.environ.get("EVAL_SEED", "987654321"))
    EVAL_SWEEP_MAX = int(os.environ.get("EVAL_SWEEP_MAX", "100"))
    EVAL_SWEEP_TRIALS = int(os.environ.get("EVAL_SWEEP_TRIALS", "1"))
    EVAL_SPAWN_POINTS = int(os.environ.get("EVAL_SPAWN_POINTS", "20"))
    EVAL_MATCH_CAP_SECONDS = int(os.environ.get("EVAL_MATCH_CAP_SECONDS", "240"))
    EVAL_MAX_JOBS = max(1, min(20, int(os.environ.get("EVAL_MAX_JOBS", "20"))))
    EVAL_ENEMY_X = float(os.environ.get("EVAL_ENEMY_X", "1920"))
    EVAL_ENEMY_Y = float(os.environ.get("EVAL_ENEMY_Y", "40"))
