import json
import logging
import urllib.request
import urllib.error
from flask import Blueprint, jsonify

version_bp = Blueprint("version", __name__, url_prefix="/api/version")
logger = logging.getLogger(__name__)

# Simple cache to avoid hitting GitHub rate limits (60/hr unauthenticated)
_cache = {
    "data": None,
    "timestamp": 0
}
CACHE_TTL = 300  # 5 minutes

@version_bp.get("")
def get_version():
    import time
    now = time.time()
    
    if _cache["data"] and (now - _cache["timestamp"]) < CACHE_TTL:
        return jsonify(_cache["data"])
        
    url = "https://api.github.com/repos/GMU-ASRC/astro-swarm/releases"
    req = urllib.request.Request(url, headers={"User-Agent": "AstroSwarm-Web"})
    
    try:
        with urllib.request.urlopen(req, timeout=5) as response:
            data = json.loads(response.read().decode("utf-8"))
            if isinstance(data, list) and len(data) > 0:
                latest_release = data[0]
                version = latest_release.get("tag_name", "")
                response_data = {"version": version}
                _cache["data"] = response_data
                _cache["timestamp"] = now
                return jsonify(response_data)
            else:
                return jsonify({"error": "No releases found"}), 404
    except urllib.error.HTTPError as e:
        if e.code == 404:
            return jsonify({"error": "No releases found"}), 404
        logger.error(f"Failed to fetch version from GitHub: {e}")
        if _cache["data"]:
            return jsonify(_cache["data"])
        return jsonify({"error": "Could not fetch release info"}), 503
    except urllib.error.URLError as e:
        logger.error(f"Failed to fetch version from GitHub: {e}")
        if _cache["data"]:
            return jsonify(_cache["data"])
        return jsonify({"error": "Could not fetch release info"}), 503
