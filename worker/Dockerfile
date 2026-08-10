FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/astroworker ./cmd/astroworker \
 && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/astrosim ./cmd/astrosim

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 astroswarm \
 && mkdir -p /data \
 && chown astroswarm:astroswarm /data

COPY --from=build /out/astroworker /usr/local/bin/astroworker
COPY --from=build /out/astrosim /usr/local/bin/astrosim

USER astroswarm
ENV WORKER_ID_FILE=/data/worker_id
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/astroworker"]
