FROM oven/bun:1-alpine AS client

WORKDIR /client

COPY package.json .npmrc ./
RUN bun install

COPY svelte.config.js vite.config.ts tsconfig.json ./
COPY src ./src
COPY static ./static

ENV PUBLIC_API_URL=""
RUN bun run build


FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg ca-certificates curl \
    && install -d /usr/share/postgresql-common/pgdg \
    && curl -fsSL -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc https://www.postgresql.org/media/keys/ACCC4CF8.asc \
    && . /etc/os-release \
    && echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt ${VERSION_CODENAME}-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update && apt-get install -y --no-install-recommends postgresql-client-16 \
    && rm -rf /var/lib/apt/lists/*

COPY server/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY server/ ./
COPY --from=client /client/build ./client

ENV PORT=5050
ENV CLIENT_DIR=/app/client

EXPOSE ${PORT}

CMD exec gunicorn --bind "0.0.0.0:${PORT}" --workers 1 --threads 8 main:app
