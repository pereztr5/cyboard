# Running postgres through docker

Short instructions on running the Cyboard postgres database (the most cumbersome
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
#
# --postgres-uri notes:
# dbname, and user are 'cybot' (dbname is implied if not specified)
# host is the db available on localhost or unix socket
go run main.go server -c cfg/config.toml --postgres-uri 'user=cybot host=/run/cyboard-db/ sslmode=disable'

# You can connect to the db with other tools in a similar fashion:
psql -h /var/run/cyboard-db/ cybot cybot  # Using a unix socket
psql -h 127.0.0.1 cybot cybot             # Over tcp/ip
psql --host 127.0.0.1 --dbname cybot --username cybot
psql postgresql://cybot@127.0.0.1/cybot
# Or with just docker:
sudo docker exec -it cyboard-db psql -U cybot

# Developers can easily restore a previous event from backup:
sudo docker cp ./data.dump cyboard-db:/tmp/
sudo docker exec cyboard-db pg_restore -U cybot /tmp/data.dump
```

_Note_: This setup cannot be used to run the tests at this time.
