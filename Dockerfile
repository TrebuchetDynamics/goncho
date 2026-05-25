# syntax=docker/dockerfile:1

FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/goncho-server ./cmd/goncho-server

FROM debian:bookworm-slim AS runtime
RUN useradd --system --uid 10001 --home-dir /data --create-home goncho
COPY --from=build /out/goncho-server /usr/local/bin/goncho-server
RUN mkdir -p /data && chown -R goncho:goncho /data
USER goncho
WORKDIR /data
EXPOSE 8765
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/usr/local/bin/goncho-server", "health", "-db", "/data/goncho.db"]
ENTRYPOINT ["/usr/local/bin/goncho-server"]
CMD ["serve", "-db", "/data/goncho.db", "-addr", "127.0.0.1:8765"]
