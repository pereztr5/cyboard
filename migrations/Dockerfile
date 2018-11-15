FROM timescale/timescaledb:1.0.0-pg10
MAINTAINER cnyhackathon
ENV        POSTGRES_USER  cybot

# Replace timescaledb upstream initdb scripts
RUN find /docker-entrypoint-initdb.d/ -type f -print -delete
COPY \
  000_docker_pgdb.up.sh \
  001_initialize_schema.up.sql \
  /docker-entrypoint-initdb.d/

