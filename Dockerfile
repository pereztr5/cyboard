FROM golang:1.9-alpine3.6 AS builder
LABEL maintainer="cnyhackathon"

ENV APP_DIR       /go/src/github.com/pereztr5/cyboard

RUN apk add --no-cache --virtual .build-deps curl git && \
    curl https://glide.sh/get | sh

WORKDIR $APP_DIR

COPY glide.* $APP_DIR/
RUN glide --home $APP_DIR/ install

COPY cmd/ $APP_DIR/cmd
COPY server/ $APP_DIR/server
COPY main.go $APP_DIR/

RUN \
    go-wrapper install -ldflags '-s -w' && \
    mv -vf  /go/bin/cyboard /srv && \
    rm -rf $APP_DIR/* && \
    apk del .build-deps

FROM alpine:3.6
RUN \
  adduser -h /srv -s /sbin/nologin -u 1000 -D cyboard && \
  apk add --no-cache dumb-init

USER cyboard
WORKDIR /srv

# Copy the binary from the build container, and the static files from source repo
COPY --from=builder /srv/cyboard /bin/cyboard
COPY static/ /srv/static
COPY tmpl/ /srv/tmpl

RUN mkdir -p /srv/log
VOLUME /srv/log/

# `docker run --entrypoint ""` will override this setting, if required (say, to examine the container).
ENTRYPOINT ["/usr/bin/dumb-init", "--", "cyboard"]
CMD ["--help"]
