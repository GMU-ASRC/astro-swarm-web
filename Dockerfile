FROM oven/bun:1-alpine AS client

WORKDIR /client

COPY package.json .npmrc ./
RUN bun install

COPY svelte.config.js vite.config.ts tsconfig.json ./
COPY src ./src
COPY static ./static

ENV PUBLIC_API_URL=""
RUN bun run build


FROM golang:1.26-alpine AS server

WORKDIR /src

COPY worker/go.mod worker/go.sum ./
RUN go mod download

COPY worker/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/astroserver ./cmd/astroserver


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata ffmpeg postgresql16-client

WORKDIR /app

COPY --from=server /out/astroserver /usr/local/bin/astroserver
COPY --from=client /client/build ./client

ENV PORT=5050
ENV CLIENT_DIR=/app/client
ENV UPLOAD_DIR=/app/uploads
ENV BACKUP_DIR=/app/backups

EXPOSE ${PORT}

ENTRYPOINT ["/usr/local/bin/astroserver"]
CMD ["serve"]
