FROM --platform=$BUILDPLATFORM golang:1.24.5-alpine AS ci

WORKDIR /workdir

# Install any dependencies that are needed to build your application.
RUN apk add --no-cache make

# Download dependencies as a separate step to take advantage of Docker's caching.
# Leverage bind mounts to go.sum, go.mod and Makefile to avoid having to copy
# them intothe container.
RUN \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=Makefile,target=Makefile \
    make -j dep

ARG ZONE
ARG TOKEN

# Test and build the application.
# Leverage bind mounts to the current directory to avoid having to copy the
# source code into the container.
RUN \
    --mount=type=bind,source=cmd,target=cmd \
    --mount=type=bind,source=internal,target=internal \    
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=Makefile,target=Makefile \
    make -j -W dep ci

FROM --platform=$BUILDPLATFORM eclipse-temurin:21.0.7_6-jre-alpine AS result

# Install bash and download and extract Codacy coverage reporter
RUN \
  apk add --no-cache bash && \
  wget -qO - https://coverage.codacy.com/get.sh > /usr/local/bin/codacy-coverage && \
  chmod +x /usr/local/bin/codacy-coverage && \
  codacy-coverage download

WORKDIR /coverage

# Copy coverage reports from CI stage
COPY --from=ci /workdir/coverage/. .

ENTRYPOINT [ "codacy-coverage" ]


FROM alpine:3.22.0 AS deploy
ARG TARGETARCH

# Set OCI image labels
ARG BUILD_DATE
LABEL \
  org.opencontainers.image.title="Hetzner DynDNS" \
  org.opencontainers.image.authors="Marvin A. Ruder <hetzner-dyndns@mruder.dev>" \
  org.opencontainers.image.description="A proxy server for updating DNS records on Hetzner DNS using the DynDNS protocol." \
  org.opencontainers.image.url="https://github.com/marvinruder/hetzner-dyndns" \
  org.opencontainers.image.source="https://github.com/marvinruder/hetzner-dyndns" \
  org.opencontainers.image.vendor="Marvin A. Ruder" \
  org.opencontainers.image.licenses="MIT" \
  org.opencontainers.image.version="0.3.0" \
  org.opencontainers.image.created=$BUILD_DATE

HEALTHCHECK CMD wget -qO /dev/null http://localhost:8245/health || exit 1

# Create a non-privileged user that the app will run under.
# See https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#user
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "1000" \
    hetzner-dyndns
USER hetzner-dyndns

# Expose the port that the application listens on.
EXPOSE 8245

# Copy the executable.
COPY --from=ci /workdir/bin/hetzner-dyndns-${TARGETARCH} /usr/local/bin/hetzner-dyndns

# What the container should run when it is started.
ENTRYPOINT [ "hetzner-dyndns" ]
