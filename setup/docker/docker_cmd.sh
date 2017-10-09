#!/bin/sh
# Sample `docker run` commands when working with
# the image produced by the Dockerfile

server() {
    # For running the web server with the CTF event:
    #   This mounts the host's ./certs and ./cfg folders, making them available (read-only) in the container.
    #   Then the app is made accessible on port 80 & 443 (HTTP & HTTPS).
    docker run -it --rm \
        -v ${PWD}/certs:/srv/certs:ro -v ${PWD}/cfg:/srv/cfg:ro \
        -p 80:8080 -p 443:8081 \
        cyboard server --config=/srv/cfg/config.toml --stdout
}

checks() {
    # Docker run can also be used this way to start the Service Checker:
    #   Since the container is a slimmed down linux, statically compiled binaries are
    #   the best options for testing scripts, as tools like wget, curl, and ssh are missing.
    docker run -it --rm \
        -v ${PWD}/scripts:/srv/scripts:ro -v ${PWD}/cfg:/srv/cfg:ro \
        cyboard checks --config=/srv/cfg/checks.toml --stdout
}

case "$1" in
server) server ;;
checks) checks ;;
     *) echo "Usage: $0 [server|checks]" ;;
esac
