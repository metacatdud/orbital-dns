FROM debian:stable-slim

# Disable interactive prompts during apt-get installs
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y resolvconf --no-install-recommends && \
    ln -sf /run/resolvconf/resolv.conf /etc/resolv.conf || true && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY ./bin/orbitaldns /app/orbitaldns
COPY ./certs/cert.pem /app/certs/cert.pem
COPY ./certs/key.pem /app/certs/key.pem

EXPOSE 53/udp
EXPOSE 443

CMD ["/app/orbitaldns"]