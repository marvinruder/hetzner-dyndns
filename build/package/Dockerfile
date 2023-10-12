# syntax=docker/dockerfile:1

FROM alpine:3.18.4

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
  org.opencontainers.image.version="0.1.3" \
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

# Copy the executable.
COPY bin/hetzner-dyndns-${TARGETARCH} /bin/hetzner-dyndns

# Expose the port that the application listens on.
EXPOSE 8245

# What the container should run when it is started.
ENTRYPOINT [ "/bin/hetzner-dyndns" ]
