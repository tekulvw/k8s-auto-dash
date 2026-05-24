# syntax=docker/dockerfile:1.7

# ---- Stage 1: UI build (Node) ---------------------------------------
FROM node:22-alpine AS ui
WORKDIR /ui

# Install bash (required by fetch-icons.mjs which uses /bin/bash) and dependencies.
RUN apk add --no-cache bash curl tar
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

COPY ui/ ./
ARG ICONS_COMMIT
RUN test -n "$ICONS_COMMIT" || (echo "ICONS_COMMIT build-arg required" && exit 1)
RUN ICONS_COMMIT=$ICONS_COMMIT pnpm icons \
 && pnpm run build

# ---- Stage 2: Go build ----------------------------------------------
FROM golang:1.26-alpine AS go
WORKDIR /src

# Dependencies (cache layer).
COPY go.mod go.sum ./
RUN go mod download

# Source.
COPY . .

# Pull UI artifacts into the embed source dirs.
RUN rm -rf internal/assets/ui/* internal/assets/icons/* \
 && mkdir -p internal/assets/ui internal/assets/icons
COPY --from=ui /ui/build/.  internal/assets/ui/
COPY --from=ui /ui/icons/.  internal/assets/icons/

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/k8s-auto-dash \
      ./cmd/k8s-auto-dash

# ---- Stage 3: Final -------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=go /out/k8s-auto-dash /k8s-auto-dash
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/k8s-auto-dash"]
