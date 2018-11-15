# Cyboard Docker Compose

With `docker-compose`, cyboard can be run without having to worry about what's
installed on the local system. [Docker][docker-overview] handles isolating the
app and dependencies into containers that can be managed easily.

## Preparation

As preparation, install [docker][docker-install] &
[docker-compose][compose-install]. Then, configure `cyboard` itself:

1. Navigate to the root of the source repo.
2. Create the directory `cfg/` and copy+modify `config.toml` in there.
3. Docker will override details such as the postgres URI & webserver ports,
   so those can be left untouched.
4. Optionally: Place an SSL cert+key into the `cfg/` and refer to them in
   `config.toml` as `/home/cyboard/.cyboard/mycert.pem` and `.../mycert.key`

## Running

Once the prep is complete, simply run:
``` sh
docker-compose -f setup/docker/docker-compose.yml up --build -d
# The `--build` flag is only required on first run or if code changes were made
```

This will build everything and start all the services/containers require to run Cyboard.
From here, you can browse to `http://localhost`, and see the app in action.

Compose mounts the host's ./data & ./cfg folders, making them available in the container.
* `./cfg`: Server & Event configuration; SSL certs.
* `./data/log/`: Log files (or `docker-compose logs` if logs sent to stdout)
* `./data/scripts/`: Service monitor commands.
   - NOTE: Use self-contained binaries, as the Docker does not cmds like
           ssh, awk, perl, ftp, mysql, etc.
* `./data/ctf/`: CTF challenge files

To shut Cyboard down:
``` sh
docker-compose -f setup/docker/docker-compose.yml down
# Append `-v` to wipe everything
```

[docker-overview]: https://docs.docker.com/engine/docker-overview/
[docker-install]: https://docs.docker.com/engine/installation/
[compose-install]: https://docs.docker.com/compose/install/
