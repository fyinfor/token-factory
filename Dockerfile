FROM oven/bun:1@sha256:0733e50325078969732ebe3b15ce4c4be5082f18c4ac1a0f0ca4839c2e4e42a7 AS builder

WORKDIR /build
COPY web/package.json .
COPY web/bun.lock .
RUN bun install
COPY ./web .
COPY ./VERSION .
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

FROM golang:1.26.2-alpine@sha256:c2a1f7b2095d046ae14b286b18413a05bb82c9bca9b25fe7ff5efef0f0826166 AS builder2
ENV GO111MODULE=on CGO_ENABLED=0

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
ENV GOEXPERIMENT=greenteagc

WORKDIR /build

RUN apk add --no-cache git ca-certificates

ADD go.mod go.sum ./
RUN --mount=type=secret,id=github_token \
    set -eux; \
    token="$(cat /run/secrets/github_token || true)"; \
    if [ -n "$token" ]; then \
      git config --global url."https://x-access-token:${token}@github.com/".insteadOf "https://github.com/"; \
    fi; \
    go env -w GOPRIVATE=github.com/fyinfor/*; \
    go mod download; \
    if [ -n "$token" ]; then \
      git config --global --unset-all url."https://x-access-token:${token}@github.com/".insteadOf || true; \
    fi

COPY . .
COPY --from=builder /build/dist ./web/dist
RUN go build -ldflags "-s -w -X 'https://github.com/fyinfor/token-factory/common.Version=$(cat VERSION)'" -o token-factory

FROM debian:bookworm-slim@sha256:f06537653ac770703bc45b4b113475bd402f451e85223f0f2837acbf89ab020a

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libasan8 wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

COPY --from=builder2 /build/token-factory /
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/token-factory"]
