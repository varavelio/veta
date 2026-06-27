# Packages the Veta CLI into a minimal, ready-to-use container image.
# The binary is fetched from GitHub Releases so this Dockerfile does not compile
# Veta. Release jobs must publish GitHub Release assets before building images.

FROM alpine:3.23.3 AS downloader

ARG TARGETARCH
ARG VETA_VERSION

RUN apk add --no-cache curl tar

RUN set -e; \
    test -n "${VETA_VERSION}" || { echo "VETA_VERSION is required" >&2; exit 1; }; \
    case "${TARGETARCH}" in \
      amd64) ARCH="amd64" ;; \
      arm64) ARCH="arm64" ;; \
      *) echo "Unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    FILENAME="veta_linux_${ARCH}.tar.gz"; \
    URL="https://github.com/varavelio/veta/releases/download/v${VETA_VERSION}/${FILENAME}"; \
    CHECKSUMS_URL="https://github.com/varavelio/veta/releases/download/v${VETA_VERSION}/checksums.txt"; \
    echo "Downloading ${URL}"; \
    curl -fsSL "${URL}" -o "/tmp/${FILENAME}"; \
    curl -fsSL "${CHECKSUMS_URL}" -o /tmp/checksums.txt; \
    cd /tmp; \
    grep "  ${FILENAME}$" checksums.txt | sha256sum -c -; \
    tar -xzf "/tmp/${FILENAME}" -C /usr/local/bin veta; \
    chmod +x /usr/local/bin/veta; \
    /usr/local/bin/veta --version

FROM debian:trixie-slim

ARG VETA_VERSION

LABEL org.opencontainers.image.title="Veta" \
      org.opencontainers.image.authors="Varavel" \
      org.opencontainers.image.description="Veta static site generator" \
      org.opencontainers.image.url="https://github.com/varavelio/veta" \
      org.opencontainers.image.documentation="https://github.com/varavelio/veta" \
      org.opencontainers.image.source="https://github.com/varavelio/veta" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="Varavel" \
      org.opencontainers.image.version="${VETA_VERSION}"

COPY --from=downloader /usr/local/bin/veta /usr/local/bin/veta

ENTRYPOINT ["veta"]
CMD ["--help"]
