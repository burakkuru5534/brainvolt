# BrainVolt backend image (roadmap 1.6.8): one image with the api, worker
# and migrate binaries. Build context: backend/.
#
#   docker build -t brainvolt/backend backend
#   docker compose --profile app up -d --build      (see docker-compose.yml)

FROM golang:1.27.2-alpine3.24 AS build
WORKDIR /src

# Dependencies first, so code changes reuse the module cache layer.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static binaries (no cgo); -trimpath keeps local paths out of the binary;
# timetzdata embeds the time-zone database (Europe/Istanbul periods, D-029).
RUN CGO_ENABLED=0 go build -trimpath -tags timetzdata -ldflags="-s -w" -o /out/ ./cmd/api ./cmd/worker ./cmd/migrate

# The runtime stage downloads nothing: CA certificates come from the build
# stage, the user is created with busybox. busybox wget serves the
# Compose health check.
FROM alpine:3.24.2
RUN addgroup -S brainvolt && adduser -S -G brainvolt brainvolt
# CA certificates: Firebase (token keys, Admin API) is reached over HTTPS.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/ /app/
USER brainvolt
EXPOSE 8080
ENV BRAINVOLT_HTTP_ADDR=:8080 BRAINVOLT_LOG_FORMAT=json
CMD ["/app/api"]
