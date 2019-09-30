# Running postgres through docker

Instructions on running the Cyboard postgres database (the most cumbersome
piece of infrastructure to set up) using a few short Docker commands:

```bash
# You'll need docker privs (be root, or in group 'docker')
su

# Build db image
( cd ./migrations && docker build -t cyboard-db . )

# Run db container with persistent, named volume for data
docker volume create pg-data
OPTS=""
OPTS+=" --name cyboard-db"
OPTS+=" --publish 127.0.0.1:5432:5432"
OPTS+=" --volume db-data:/var/lib/postgresql/data"
OPTS+=" --volume /run/cyboard-db/:/run/postgresql/"
docker run --detach --rm $OPTS cyboard-db

# Drop back to normal user
exit

# Run cyboard server.
go run main.go server -c cfg/config.toml --postgres-uri 'host=/run/cyboard-db/ dbname=postgres user=cyboard host=/run/cyboard-db/ sslmode=disable'


# You can connect to the db with other tools in a similar fashion:
psql -h /var/run/cyboard-db/ postgres cyboard  # Using a unix socket
psql -h 127.0.0.1 postgres cyboard             # Over tcp/ip
psql --host 127.0.0.1 --dbname postgres --username cyboard
psql postgresql://cyboard@127.0.0.1/postgres
# Or with just docker:
sudo docker exec -it cyboard-db psql -U cyboard -d postgres


# Developers can restore a previous event from backup:
sudo docker cp ./data.dump cyboard-db:/tmp/data.dump
sudo docker exec cyboard-db psql -X -U postgres -c 'SELECT timescaledb_pre_restore();' -c '\! pg_restore --clean -Fc -U postgres -d postgres /tmp/data.dump;' -c 'SELECT timescaledb_post_restore();'
# (this can produce hundreds of "already exists" errors, which can be ignored, but sometimes it does fail unexpectedly.)


# To run the go tests (readme's "Testing" section), pass two extra environment variables:
OPTS=" -e CYTEST=t -e POSTGRES_DB=cyboard_test"
OPTS+=" --name cyboard-db -p 127.0.0.1:5432:5432"
sudo docker run --detach --rm $OPTS cyboard-db
go test -p 1 ./...
```
