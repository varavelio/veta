#########
# TOOLS #
#########

# Use golang (debian) as the base image
FROM golang:1.27.2-trixie as tools

# Install node.js
COPY --from=node:26.11.1-trixie-slim /usr/local /usr/local

# Install golangci-lint
COPY --from=golangci/golangci-lint:v2.12.2 /usr/bin/golangci-lint /usr/local/bin/golangci-lint

# Install dprint
COPY --from=ghcr.io/varavelio/container-tools/dprint:0.63.0 /usr/local/bin/dprint /usr/local/bin/dprint

# Install task
COPY --from=ghcr.io/varavelio/container-tools/task:3.54.0 /usr/local/bin/task /usr/local/bin/task

# Install current version of veta as veta-prev
COPY --from=varavel/veta:0.1.1 /usr/local/bin/veta /usr/local/bin/veta-prev

# Environment variables
ENV \
  PIP_BREAK_SYSTEM_PACKAGES=1 \
  CGO_ENABLED=0

# Build time variables
ARG DEBIAN_FRONTEND=noninteractive

RUN set -e && \
  # Update and install system dependencies
  apt-get update -qq && \
  apt-get install -yqq --no-install-recommends \
  ca-certificates wget curl zip unzip p7zip-full tzdata git tree ripgrep \
  python3 python3-pip && \
  rm -rf /var/lib/apt/lists/* && \
  # Git config
  git config --system --add safe.directory '*'

WORKDIR /workspaces/veta

################
# DEVCONTAINER #
################

FROM tools AS devcontainer

CMD ["sleep", "infinity"]
