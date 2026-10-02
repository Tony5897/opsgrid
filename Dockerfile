# syntax=docker/dockerfile:1

# One image, four binaries (api, worker, relay, migrate) + healthcheck.
# Compose and deployments select the binary with `command`.

ARG GO_VERSION=1.27.1
ARG NODE_VERSION=24

# ---- web: build the SPA ------------------------------------------------------
FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS web
WORKDIR /src
ARG PNPM_VERSION=12.8.1
RUN npm install -g pnpm@${PNPM_VERSION}
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc ./
COPY apps/web/package.json apps/web/
COPY packages/api-client/package.json packages/api-client/
COPY tools/openapi-gen/package.json tools/openapi-gen/
RUN --mount=type=cache,id=pnpm,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile
COPY apps/web apps/web
COPY packages packages
RUN pnpm --filter @opsgrid/web build

# ---- go: compile all binaries with the SPA embedded ---------------------------
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-trixie AS go
WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local GOFLAGS=-mod=readonly
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd cmd
COPY db db
COPY internal internal
COPY --from=web /src/apps/web/dist internal/platform/webui/dist
ARG TARGETOS TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
      -ldflags "-s -w \
        -X github.com/Tony5897/opsgrid/internal/platform/buildinfo.Version=${VERSION} \
        -X github.com/Tony5897/opsgrid/internal/platform/buildinfo.Commit=${COMMIT}" \
      -o /out/ ./cmd/api ./cmd/worker ./cmd/relay ./cmd/migrate ./cmd/healthcheck

# ---- runtime -------------------------------------------------------------------
FROM gcr.io/distroless/static-debian13:nonroot
LABEL org.opencontainers.image.source="https://github.com/Tony5897/opsgrid" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.title="opsgrid"
COPY --from=go /out/ /usr/local/bin/
USER nonroot:nonroot
EXPOSE 8080 9090
HEALTHCHECK --interval=10s --timeout=4s --start-period=10s --retries=3 \
  CMD ["/usr/local/bin/healthcheck", "http://127.0.0.1:9090/readyz"]
ENTRYPOINT ["/usr/local/bin/api"]
