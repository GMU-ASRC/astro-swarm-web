import logging
import os
import shutil
import zipfile

import requests

logger = logging.getLogger("worker.godot")

REPO = os.environ.get("GODOT_RELEASE_REPO", "GMU-ASRC/astro-swarm")
TAG = os.environ.get("GODOT_RELEASE_TAG", "latest")
ASSET = os.environ.get("GODOT_RELEASE_ASSET", "AstroSwarm_Linux_Server.zip")
DEST_DIR = os.environ.get("GODOT_DIR", "/data/server_build")

BINARY_SUFFIXES = (".x86_64", ".x86_32", ".arm64", ".arm32")
MARKER = ".release"


def _release_url():
    if TAG == "latest":
        return f"https://api.github.com/repos/{REPO}/releases/latest"
    return f"https://api.github.com/repos/{REPO}/releases/tags/{TAG}"


def _api_headers():
    headers = {"Accept": "application/vnd.github+json"}
    token = os.environ.get("GITHUB_TOKEN")
    if token:
        headers["Authorization"] = f"Bearer {token}"
    return headers


def _candidate_files(root):
    files = []
    for dirpath, _dirs, names in os.walk(root):
        for name in names:
            if name.startswith(".") or name.endswith((".pck", ".zip")):
                continue
            files.append(os.path.join(dirpath, name))
    return files


def _find_binary(root):
    files = _candidate_files(root)
    for path in files:
        if path.endswith(BINARY_SUFFIXES):
            return path
    if not files:
        return None
    return max(files, key=lambda p: os.path.getsize(p))


def _read_marker():
    path = os.path.join(DEST_DIR, MARKER)
    if os.path.isfile(path):
        with open(path) as f:
            return f.read().strip()
    return None


def ensure_server_build():
    """Resolve the Godot dedicated-server binary. If GODOT_SERVER_BIN points at an
    existing file it is used as-is; otherwise the release asset is downloaded and
    unzipped into GODOT_DIR (re-downloaded only when the release tag changes).
    Returns (binary_path, pck_path or None). The downloaded export binary loads its
    adjacent .pck automatically, so no pack path override is returned for it."""
    explicit = os.environ.get("GODOT_SERVER_BIN")
    if explicit and os.path.isfile(explicit):
        logger.info("using provided GODOT_SERVER_BIN=%s", explicit)
        return explicit, (os.environ.get("GODOT_PCK") or None)

    resp = requests.get(_release_url(), headers=_api_headers(), timeout=30)
    resp.raise_for_status()
    release = resp.json()
    tag = release.get("tag_name", "unknown")

    existing = _find_binary(DEST_DIR)
    if _read_marker() == tag and existing:
        logger.info("server build %s already present at %s", tag, existing)
        return existing, None

    asset = next((a for a in release.get("assets", []) if a.get("name") == ASSET), None)
    if asset is None:
        raise RuntimeError(f"release {tag} has no asset named {ASSET}")

    if os.path.isdir(DEST_DIR):
        shutil.rmtree(DEST_DIR)
    os.makedirs(DEST_DIR, exist_ok=True)

    zip_path = os.path.join(DEST_DIR, ASSET)
    logger.info("downloading %s from %s release %s", ASSET, REPO, tag)
    with requests.get(asset["browser_download_url"], headers=_api_headers(), stream=True, timeout=900) as download:
        download.raise_for_status()
        with open(zip_path, "wb") as f:
            for chunk in download.iter_content(chunk_size=1 << 20):
                f.write(chunk)

    with zipfile.ZipFile(zip_path) as archive:
        archive.extractall(DEST_DIR)
    os.remove(zip_path)

    binary = _find_binary(DEST_DIR)
    if binary is None:
        raise RuntimeError(f"no server binary found inside {ASSET}")
    os.chmod(binary, 0o755)

    with open(os.path.join(DEST_DIR, MARKER), "w") as f:
        f.write(tag)

    logger.info("server build %s ready: binary=%s", tag, binary)
    return binary, None
