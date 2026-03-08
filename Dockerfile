FROM golang:1.26.0 AS builder

WORKDIR /app
COPY api .
RUN go build -o app .

FROM ghcr.io/linuxserver/chromium:version-09bef544

RUN \
    echo "**** install packages ****" && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    scrot && \
    echo "**** cleanup ****" && \
    apt-get autoclean && \
    rm -rf \
    /config/.cache \
    /var/lib/apt/lists/* \
    /var/tmp/* \
    /tmp/*

COPY --from=builder /app/app /usr/local/bin/api
RUN chmod +x /usr/local/bin/api

RUN mkdir -p /etc/s6-overlay/s6-rc.d/api \
    && mkdir -p /etc/s6-overlay/s6-rc.d/user/contents.d

COPY api/s6/type /etc/s6-overlay/s6-rc.d/api/type
COPY api/s6/run /etc/s6-overlay/s6-rc.d/api/run

RUN chmod +x /etc/s6-overlay/s6-rc.d/api/run \
    && touch /etc/s6-overlay/s6-rc.d/user/contents.d/api