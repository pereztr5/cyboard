FROM timescale/timescaledb:1.4.2-pg11
MAINTAINER cnyhackathon

# Add user & schema setup scripts. Filenames are all numbered in the form "XXXcy_*",
# so as not to collide with the timescaledb base docker's scripts.
COPY --chown=postgres:postgres \
  000cy_docker_pgdb.sh \
  001cy_user_setup.up.sql \
  002cy_initialize_schema.up.sql \
  /docker-entrypoint-initdb.d/

