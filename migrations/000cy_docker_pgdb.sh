#!/bin/sh
#
# Only used in the docker build to bootstrap the PG database.
#

# Set the search_path for all users on all databases to include the cyboard schema.
# Takes effect on restart of postgres (which the container does after all scripts run).
psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
  -c 'ALTER SYSTEM SET search_path = cyboard, "$user", public;'

if [ -n "${CYTEST}" ]; then
    echo "Setting up DB for testing..."
    createuser -U "${POSTGRES_USER}" --superuser --login --echo supercyboard
fi
