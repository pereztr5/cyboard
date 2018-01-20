# Cyboard Docker Compose

With `docker-compose`, cyboard can be run without having to worry about what's
installed on the local system. [Docker][docker-overview] handles isolating the
app and everything is installed in a virtual environment with one simple
command.

## Preparation

As preparation, install [docker][docker-install] &
[docker-compose][compose-install]. Then, configure `cyboard` itself:

1. Copy the necessary files from here to the root of the project:
   `$ cp .dockerignore docker-compose.yml Dockerfile ../../`
2. Create the directory `cfg/` at the root.
3. Copy `config.toml` and `checks.toml` into the new folder.
4. Update their settings as you see fit. Docker will handle details such as
   the mongodb URI, & webserver ports, so those can be left untouched.
5. Optionally: Create SSL certs using the `setup/generate_cert.go` script,
   and place those in a `certs/` folder at the repo root.
6. Optionally: Create a `scripts/` folder for checks, and add scripts/commands
   as needed by `checks.toml`.

## Running

Once the prep is complete, simply run:
``` sh
docker-compose --build -d up
# The `--build` flag is only required on first run or if code changes were made
```

This will build everything and start all the services require to run Cyboard.
From here, you can browse to `http://localhost`, and see the app in action.

To shut Cyboard down:
``` sh
docker-compose down
```

[docker-overview]: https://docs.docker.com/engine/docker-overview/
[docker-install]: https://docs.docker.com/engine/installation/
[compose-install]: https://docs.docker.com/compose/install/
