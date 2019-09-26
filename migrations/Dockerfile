FROM timescale/timescaledb:0.11.0-pg10
MAINTAINER cnyhackathon

# Set the main user
# DB Name becomes the same as POSTGRES_USER, since POSTGRES_DATABASE isn't specified
ENV  POSTGRES_USER  cybot

# Replace timescaledb upstream initdb scripts
RUN find /docker-entrypoint-initdb.d/ -type f -print -delete
COPY --chown=postgres:postgres \
  000_docker_pgdb.up.sh \
  001_initialize_schema.up.sql \
  /docker-entrypoint-initdb.d/

