# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.25-bookworm AS build

WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Build the static-ish binary.
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
    -o /usr/local/bin/x-article-exporter .

# ---- Runtime stage ----
FROM debian:bookworm-slim AS runtime

# Pin the Typst release used for PDF rendering.
ARG TYPST_VERSION=v0.13.1

# Install fonts + CA certs, then download the official Typst binary for the
# target architecture.
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        xz-utils \
        fonts-dejavu \
        fonts-noto-core \
        fonts-noto-cjk; \
    arch="$(dpkg --print-architecture)"; \
    case "$arch" in \
        amd64) typst_arch="x86_64" ;; \
        arm64) typst_arch="aarch64" ;; \
        *) echo "unsupported architecture: $arch" >&2; exit 1 ;; \
    esac; \
    typst_tarball="typst-${typst_arch}-unknown-linux-musl"; \
    url="https://github.com/typst/typst/releases/download/${TYPST_VERSION}/${typst_tarball}.tar.xz"; \
    curl -fsSL "$url" -o /tmp/typst.tar.xz; \
    tar -xJf /tmp/typst.tar.xz -C /tmp; \
    install -m 0755 "/tmp/${typst_tarball}/typst" /usr/local/bin/typst; \
    rm -rf /tmp/typst.tar.xz "/tmp/${typst_tarball}"; \
    apt-get purge -y --auto-remove curl xz-utils; \
    rm -rf /var/lib/apt/lists/*; \
    typst --version

COPY --from=build /usr/local/bin/x-article-exporter /usr/local/bin/x-article-exporter

ENTRYPOINT ["x-article-exporter"]
