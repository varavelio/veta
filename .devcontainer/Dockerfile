# =============================================================================
# Shared Multi-Arch Development Image
# =============================================================================
#
# This image bundles every build time dependency required by the project and is
# reused by the devcontainer, CI pipeline, and other containers.
#
# Rebuilds only when tool versions or system packages change.
# Published to the github container registry as:
#
#   ghcr.io/varavelio/veta-dev:sha-<short-commit>
#
# -----------------------------------------------------------------------------
# Usage
# -----------------------------------------------------------------------------
#
# Refer to this image in downstream Dockerfiles:
#
#   FROM ghcr.io/varavelio/veta-dev:sha-<tag> AS builder
#
# After bumping tool versions in this file, push to main, the pipeline
# publishes the new image automatically. You may then update the tag in
# every downstream Dockerfile manually when ready.
#
# You can also use workflow_dispatch in github actions to trigger a manual
# build and publish of this image, which is useful for testing changes before
# merging to main.
# =============================================================================

# Use golang as the base image
FROM golang:1.26-trixie

# Install Node.js
COPY --from=node:24-trixie /usr/local/ /usr/local/

# Install golangci-lint
COPY --from=golangci/golangci-lint:v2.12.2 /usr/bin/golangci-lint /usr/local/bin/golangci-lint

# Install previous version of veta as veta-prev
COPY --from=varavel/veta:0.1.0-alpha.2 /usr/local/bin/veta /usr/local/bin/veta-prev

# Set environment variables
ENV \
  DEBIAN_FRONTEND="noninteractive" \
  PIP_BREAK_SYSTEM_PACKAGES=1

RUN set -e && \
  # Update and install system dependencies
  apt-get update -qq && \
  apt-get install -yqq --no-install-recommends \
  ca-certificates wget curl zip unzip p7zip-full tzdata git tree ripgrep \
  python3 python3-pip && \
  # Install tools from npm
  npm install --global @go-task/cli@3.51.1 && \
  npm install --global dprint@0.54.0 && \
  # Git config
  git config --global --add safe.directory '*' && \
  # Cleanup
  npm cache clean --force && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Just to keep the container running
CMD ["sleep", "infinity"]
