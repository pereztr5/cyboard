#!/bin/bash
#
# Only used in the docker build to bootstrap the PG database.
#

# Copy setup steps from upstream timescaledb docker images

# respect telemetry settings
TS_TELEMETRY='basic'
if [ "${TIMESCALEDB_TELEMETRY}" == "off" ]; then
	TS_TELEMETRY='off'
fi

echo "timescaledb.telemetry_level=${TS_TELEMETRY}" >> ${PGDATA}/postgresql.conf

