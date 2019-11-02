FROM golang:1.13-alpine AS builder
LABEL maintainer="cnyhackathon"

ENV APP_DIR  /go/src/github.com/pereztr5/cyboard

WORKDIR $APP_DIR

COPY go.mod go.sum main.go  $APP_DIR/
COPY cmd    $APP_DIR/cmd
COPY server $APP_DIR/server

RUN \
    go install -ldflags '-s -w' -v ./... && \
    go clean -r -cache -modcache

FROM alpine:3.10
RUN \
  apk add --no-cache dumb-init && \
  wget https://raw.githubusercontent.com/eficode/wait-for/master/wait-for && \
  chmod a+x wait-for && \
  mv -v wait-for /usr/local/bin/ && \
  adduser -s /sbin/nologin -u 1000 -D cyboard && \
  mkdir -p /cyboard/data/log /cyboard/data/scripts /cyboard/data/ctf

USER cyboard
WORKDIR /cyboard

# Copy the binary from the build container, and the static files from source repo
COPY --from=builder /go/bin/cyboard /usr/local/bin/cyboard
COPY ui/ /cyboard/ui

VOLUME /cyboard/data/

# `docker run --entrypoint ""` will override this setting, if required (say, to examine the container).
ENTRYPOINT ["/usr/bin/dumb-init", "--", "cyboard"]
CMD ["--help"]
