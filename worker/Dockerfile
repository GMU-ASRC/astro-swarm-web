FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    libx11-6 libxcursor1 libxinerama1 libxi6 libxrandr2 libxext6 libxrender1 \
    libgl1 libglu1-mesa \
    libfontconfig1 libfreetype6 \
    libasound2 libpulse0 libudev1 \
    && rm -rf /var/lib/apt/lists/*

ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=compute,utility,video

COPY worker/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY worker/ ./
# Exported AstroSwarm dedicated-server build (binary + .pck) used to run the
# FARP benchmark. Drop your build into web/server_build/ before building.
COPY server_build/ ./server_build/
RUN chmod +x /app/server_build/AstroSwarm_Linux.x86_64 2>/dev/null || true

ENV GODOT_SERVER_BIN=/app/server_build/AstroSwarm_Linux.x86_64
ENV WORKER_ID_FILE=/data/worker_id

CMD ["python", "worker.py"]
