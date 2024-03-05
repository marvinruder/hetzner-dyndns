# Lists all images to be prefetched by the regular Docker builder
FROM scratch as prefetch
COPY --from=golang:1.22.1-alpine /etc/os-release /etc/os-release-golang
COPY --from=eclipse-temurin:21.0.2_13-jre-alpine /etc/os-release /etc/os-release-java


FROM golang:1.22.1-alpine as run

WORKDIR /src

# Install any dependencies that are needed to build your application.
# Leverage a cache mount to /var/cache/apk/ to speed up subsequent builds.
RUN --mount=type=cache,target=/var/cache/apk \
    mkdir -p /coverage /extract && \
    apk --update add \
        make \
        upx

# Download dependencies as a separate step to take advantage of Docker's caching.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage bind mounts to go.sum and go.mod to avoid having to copy them into
# the container.
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=Makefile,target=Makefile \
    make -j dep

ARG ZONE
ARG TOKEN

# Test and build the application.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage a bind mount to the current directory to avoid having to copy the
# source code into the container.
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    make -j -W dep ci

FROM eclipse-temurin:21.0.2_13-jre-alpine as result

# Install bash and download and extract Codacy coverage reporter
RUN --mount=type=cache,target=/var/cache/apk \
    mkdir -p /coverage /extract && \
    apk --update add \
        bash \
    && \
  wget -qO - https://coverage.codacy.com/get.sh > /usr/local/bin/codacy-coverage && \
  chmod +x /usr/local/bin/codacy-coverage && \
  codacy-coverage download

WORKDIR /coverage

# Copy coverage reports and binaries from run stage
COPY --from=run /coverage/. /coverage/.
COPY --from=run /extract/. /extract/.

ENTRYPOINT [ "codacy-coverage" ]


FROM alpine:3.19.1 as deploy
ARG TARGETARCH

# Set OCI image labels
LABEL \
  org.opencontainers.image.title="Hetzner DynDNS" \
  org.opencontainers.image.authors="Marvin A. Ruder <hetzner-dyndns@mruder.dev>" \
  org.opencontainers.image.description="A proxy server for updating DNS records on Hetzner DNS using the DynDNS protocol." \
  org.opencontainers.image.url="https://github.com/marvinruder/hetzner-dyndns" \
  org.opencontainers.image.source="https://github.com/marvinruder/hetzner-dyndns" \
  org.opencontainers.image.vendor="Marvin A. Ruder" \
  org.opencontainers.image.licenses="MIT" \
  org.opencontainers.image.version="0.1.5" \
  org.opencontainers.image.created=$BUILD_DATE

# Create a non-privileged user that the app will run under.
# See https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#user
ARG UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser
USER appuser

# Expose the port that the application listens on.
EXPOSE 8245

### <- This is a special marker, everything in this stage above it can and will be built and cached before the production bundle is available

# Copy the executable.
COPY bin/hetzner-dyndns-${TARGETARCH} /usr/local/bin/hetzner-dyndns

# What the container should run when it is started.
ENTRYPOINT [ "hetzner-dyndns" ]
