FROM debian:stable-slim

# Install minimal dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

ENV args=""

# Copy rootfs and wrapper binary
COPY ./rootfs /app/rootfs
COPY ./wrapper /app/wrapper

WORKDIR /app

# Make wrapper executable
RUN chmod +x /app/wrapper

# Expose ports:
# 10020 = decrypt port
# 20020 = m3u8 port
# 30020 = account port
EXPOSE 10020 20020 30020

# Healthcheck: verify wrapper is listening
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
    CMD ss -tlnp | grep -q 10020 || exit 1

CMD ["bash", "-c", "/app/wrapper $args"]
