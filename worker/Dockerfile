FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    libx11-6 libxcursor1 libxinerama1 libxi6 libxrandr2 libxext6 libxrender1 \
    libgl1 libglu1-mesa libgl1-mesa-dri \
    libvulkan1 mesa-vulkan-drivers \
    libfontconfig1 libfreetype6 \
    libasound2 libpulse0 libudev1 \
    && rm -rf /var/lib/apt/lists/*

# "graphics" is required for the GPU to expose Vulkan/OpenGL when you do pass an
# NVIDIA GPU into the container (uncomment the deploy block in docker-compose).
ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=graphics,compute,utility,video
# Fall back to CPU rendering (Mesa lavapipe / llvmpipe) when the worker host has
# no GPU passed in, so the headless Godot build can still create a rendering device.
ENV LIBGL_ALWAYS_SOFTWARE=true

COPY worker/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY worker/ ./

# The worker downloads the AstroSwarm dedicated-server build from the GitHub
# release on startup and unzips it under GODOT_DIR. Set GODOT_SERVER_BIN to a
# mounted binary instead if you want to provide your own build.
ENV GODOT_DIR=/data/server_build
ENV WORKER_ID_FILE=/data/worker_id

CMD ["python", "worker.py"]
