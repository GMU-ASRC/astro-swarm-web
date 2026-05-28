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

COPY server/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY server/ ./
COPY --from=client /client/build ./client

ENV PORT=5050
ENV CLIENT_DIR=/app/client

EXPOSE ${PORT}

CMD exec gunicorn --bind "0.0.0.0:${PORT}" --workers 4 --preload main:app
