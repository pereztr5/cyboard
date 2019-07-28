# Running postgres through docker

Short instructions on running the Cyboard postgres database (the most cumbersome
piece of infrastructure to set up) using a few short Docker commands:

```bash
# You'll need docker privs (be root, or in group 'docker')
su -

# Build db image
( cd ./migrations && docker build -t cyboard-db . )

# Run db container with persistent, named volume for data
docker volume create db-data
docker run --detach --rm -p 5432:5432 --volume db-data:/var/lib/postgresql/data cyboard-db

# Drop back to normal user
exit

# Run cyboard server.
#
# --postgres-uri notes:
# user and dbname are 'cybot' (dbname is implied if not specified)
# host is the db available on localhost
go run main.go server -c cfg/config.toml --postgres-uri 'user=cybot host=127.0.0.1 sslmode=disable'

# You can connect to the db with other tools in a similar fashion:
psql -h 127.0.0.1 cybot cybot
psql --host 127.0.0.1 --dbname cybot --username cybot 
psql postgresql://cybot@127.0.0.1/cybot
```

_Note_: This setup cannot be used to run the tests at this time.
