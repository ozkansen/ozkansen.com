FROM golang:1.27.0-trixie AS tools-templ

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go install github.com/a-h/templ/cmd/templ@latest


FROM golang:1.27.0-trixie AS stage-1
WORKDIR /app

COPY --from=tools-templ /go/bin/templ /usr/local/bin/templ
COPY . .

RUN templ generate

FROM node:24-trixie AS stage-2
WORKDIR /app

COPY --from=stage-1 /app /app
RUN npm install

RUN npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --minify


FROM golang:1.27.0-trixie AS stage-3
WORKDIR /app

COPY --from=stage-2 /app /app
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOEXPERIMENT='simd,newinliner'

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -ldflags='-w -s' -trimpath -o ./dist/server ./cmd/web

RUN cp -r static ./dist/static && rm -rf ./dist/static/css/input.css

FROM debian:trixie-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=stage-3 /app/dist/server .
COPY --from=stage-3 /app/dist/static ./static

USER 1000:1000

EXPOSE 8080

CMD ["./server"]
