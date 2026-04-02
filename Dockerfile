FROM golang:1.26.0 AS builder

WORKDIR /app
COPY api .
RUN go build -o app .

FROM ghcr.io/linuxserver/chromium:version-09bef544

# Node.js installation
ENV NODE_VERSION=v24.14.1
ENV DISTRO=linux-x64

RUN apt-get update && \
    apt-get install -y --no-install-recommends xz-utils wget scrot && \
    # Install Node.js
    wget -q https://nodejs.org/dist/$NODE_VERSION/node-$NODE_VERSION-$DISTRO.tar.xz && \
    mkdir -p /usr/local/lib/nodejs && \
    tar -xJf node-$NODE_VERSION-$DISTRO.tar.xz -C /usr/local/lib/nodejs && \
    rm node-$NODE_VERSION-$DISTRO.tar.xz && \
    # Remove build-only tools
    apt-get purge -y --auto-remove xz-utils wget && \
    apt-get autoclean && \
    rm -rf /config/.cache /var/lib/apt/lists/* /var/tmp/* /tmp/*

ENV PATH=/usr/local/lib/nodejs/node-$NODE_VERSION-$DISTRO/bin:$PATH

# MCP installation
RUN npm install -g chrome-devtools-mcp@0.21.0

# Api installation
COPY --from=builder /app/app /usr/local/bin/api
RUN chmod +x /usr/local/bin/api

# s6 service setup
RUN mkdir -p /etc/s6-overlay/s6-rc.d/api \
    /etc/s6-overlay/s6-rc.d/user/contents.d

COPY api/s6/type /etc/s6-overlay/s6-rc.d/api/type
COPY api/s6/run  /etc/s6-overlay/s6-rc.d/api/run

RUN chmod +x /etc/s6-overlay/s6-rc.d/api/run \
    && touch /etc/s6-overlay/s6-rc.d/user/contents.d/api
