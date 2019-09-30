# Cyboard

**Cyboard** is a scoring engine for the cyber defense competition
[CNY Hackathon](https://www.cnyhackathon.org "CNY Hackathon Home").

<!-- Generated with https://github.com/jonschlinkert/markdown-toc -->
<!-- toc -->

- [Background](#background)
- [Features](#features)
  * [Web Application Server](#web-application-server)
  * [Service Checker](#service-checker)
- [Building](#building)
- [Setup DB](#setup-db)
  * [Install PostgreSQL](#install-postgresql)
  * [Install Timescaledb Extension](#install-timescaledb-extension)
  * [Database Instance Setup](#database-instance-setup)
  * [Create Tables (Migrations)](#create-tables-migrations)
- [Testing](#testing)
- [Administration](#administration)
  * [CTF Event and Web Server](#ctf-event-and-web-server)
    + [Users and Roles](#users-and-roles)
    + [Running the Web Server](#running-the-web-server)
  * [Service Monitor](#service-monitor)
    + [Writing Service Checking Scripts](#writing-service-checking-scripts)
    + [Running the Service Monitor](#running-the-service-monitor)
  * [Scheduled Event Start, End and Intermissions](#scheduled-event-start-end-and-intermissions)
  * [PostgreSQL](#postgresql)
- [Docker](#docker)

<!-- tocstop -->

## Background

CNY Hackathon is a joint cyber security defense & CTF event for intercollegiate
students in the US North East region. The event is hosted bi-annually to
100+ contestants. Despite the name, it shares no similarities with a
[programming hackathon](https://en.wikipedia.org/wiki/Hackathon).

[Tony](https://github.com/pereztr5 "Tony Perez") first developed Cyboard in 2016
as his senior project at SUNY Polytechnic. Ever since Tony's graduation,
[Butters](https://github.com/tbutts "Tyler Butters") stepped up as the project's
developer.

## Features
### Web Application Server

- Fast, self-contained (no internet required) web site powered by Bootstrap
- Scoreboard display (updates automatically), shows points &
    feedback on teams' service statuses
- Locally run CTF event
    - Flag submission forms with instant feedback for contestants
    - Challenges divided into groups (e.g. Reversing, Programming, Crytpo, etc.)
    - Markdown descriptions (inline images, links, code blocks, text styles)
    - Host any custom files (crackme binaries, stego images, crypto messages)
- Web Admin Panels for User/Team, CTF, and Services
- JSON-based HTTP REST API

### Service Checker
- Scores contestants' infrastructure at regular intervals
- Checks are any script/program, language agnositc
- Completely automated during the event

-----

## Building

Cyboard is written in [Golang][golang], and has been tested on recent versions of MacOS,
FreeBSD, CentOS, Arch, and Ubuntu. To build `cyboard`:

1. Install Golang
    * The simplest method is to use your system's package manager
      to download the `golang` package - e.g. `yum install golang`
    * Alternatively Download & Install [Go v1.9+][go-install]
2. _Optional_: Go demands all code be located in one central folder,
    which you may configure before proceeding: [Guide to GOPATH][gopath]
3. Install [dep][dep], which manages Go dependencies
    * Two line install for linux:
        ``` bash
        wget -O $GOPATH/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64
        chmod 755 $GOPATH/bin/dep
        ```
4. Clone this source repo into `$GOPATH/src/github.com/pereztr5/cyboard/`
5. From the source root: `dep ensure && go build`
6. The result generated is a single, runnable binary, `cyboard`!

You can install the binary itself anywhere, with the only caveat that
the `ui/` folder from this source repo must be in the current working directory,
along with a modified copy of `config.toml`, editted to your needs.

## Setup DB

To run `cyboard`, you must have PostgreSQL installed with the
[timescaledb][timescale-home] extension. Timescale provides time-series
functionality for Postgres beyond what PG does out of the box. We use this
for analytics and keeping the DB fast.

We have used PostgreSQL **v10.3 & v10.6** with timescale **v0.11**.
Other versions of both (particularly newer ones) should work, as long as
the db and extension are compatible.

### Install PostgreSQL

PostgreSQL can be installed through your distro's package manager.
Search for it with your pkg manager, or refer to the [documentation][postgres-dl]
(note that, for instance, centos and ubuntu have separate pkg repos you will
need to enable).

You will likely need to install any package like **"postgresql10-contrib"** and
**"postgresql10-devel"**, to support the necessary extensions for cyboard. Two
of the extensions, `moddatetime` and `tablefunc` are maintained by the Postgres
team, which are usually in that "*-contrib", or just in the base "postgres-server"
package. The last extension, `timescaledb`, is made by a separate company and
cyboard uses the open-source version of it.

### Install Timescaledb Extension

[See the docs][timescale-install]. We always compile our Timescale ext from
source - it's a handful of commands, and only requires postgres devel, cmake,
and a C compiler. Installing from yum/apt repo is also easy enough.

Make sure you edit **postgresql.conf** according to the documentation.

### Database Instance Setup

First thing, if you haven't setup any other postgres instance again, you'll need
to initalize a db cluster. This differs a bit between distros, but the steps are
basically:

1. identify the default data dir ($PGDATA)
2. ensure that dir exists and is owned by the `postgres` user
3. as the user `postgres`, run:
`initdb --locale en_US.UTF-8 --encoding UTF8 --pgdata <path_to_datadir>`

As an example of differences, in CentOS, the above steps are instead to run
one command: `postgresql-setup initdb`

From here, you have a db cluster with one 'database', which is named 'postgres'.
The user and dbname are shared, and the 'postgres' user has superuser privileges.
To do anything right now, you'll need to interact with the db cluster as
that 'postgres' user, which means sudo/su your command shell before running
commands.

To make it easier, you may want to [edit the pg_hba.conf file][postgres-hba] to
allow local connections to login to the database as any user (first example
shown). With this, you can run postgres commands (`createuser`, `psql`,
`createdb`) as the 'postgres' superuser (or as the user we'll create later) by
adding the common flag: `-U <user>`.

For added security on cyboard, we use a **plain user**, `cyboard`, to run
the application. Create the user for the whole db cluster with:
`createuser -U postgres --login --echo cyboard`

Optionally, you may create a unique database in the cluster for cyboard:
`createdb -U postgres cyboard`. This isn't necessary though, as all data will be
namespaced into a unique schema to prevent any clashes with existing tables.

### Create Tables (Migrations)

Finally, to populate the database, you have to run all the migration scripts.
Currently, the easiest way to do that is a shell for-loop:

```bash
for sql in ./migrations/*.up.sql; do
    psql -U postgres --dbname cyboard -f $sql || { echo "ERR during '$sql'" && break }
done
```

## Testing

Cyboard has a modest suite of tests that can verify the program and DB work.
The tests require the full Postgres setup above, running on the local system.
Additionally, you will need to create a test admin user, test database, and
rerun the migrations:

```bash
createdb cyboard_test
createuser --superuser supercyboard
psql -U postgres -c 'ALTER ROLE supercyboard SET search_path = cyboard, public'
for sql in ./migrations/*.up.sql; do
    psql -U postgres --dbname cyboard_test -f $sql || { echo "ERR during '$sql'" && break; }
done
```

After, to run the tests: `go test -p 1 ./...`

If a test fails, check the output and make sure the issue isn't with your setup
before reporting a bug.

-----

## Administration

Cyboard is designed as two separate components, that are kicked off with
the `cyboard` command:

1. `cyboard server`: Web Application & CTF Server
2. `cyboard checks`: Service Checker / Monitor

The reasoning behind having separate services like this has been that each
piece can be distributed onto different machines, restarted independently
if one were to kick the bucket, and developed in tandem.

Cyboard's settings are managed through a single config file, and through
an admin web ui.

The config file, `config.toml`, is written in [TOML][toml], which is like
a souped up INI format. Cyboard will look for the config in either the current
working dir, your home dir under `$HOME/.cyboard/`, or you can specify the
location with `-c` or `--config` flag.

After the web server is running, everything else is configured through the
admin web ui. The dashboard has instructions on each page.

By default, all logs will be stored in `${PWD}/data/log/`. Check the `server.log`
if you're experiencing issues at start up.

### CTF Event and Web Server

> Config file: `config.toml` -> `[server]` section

> Logs: `requests.log` (HTTP Requests) :: `captured_flags.log` (Tracks flag
         submissions) :: `server.log` (Other general logs)

The CTF component is a web app where the event's scoreboard and CTF challenges
are hosted. In a computer CTF event, participants are tasked with finding flags
by solving puzzles that test their knowledge. The harder the challenge, the more
points a flag is worth. The flags themselves are little codes, such as `flag
{tHi5Fl4gBit3s!}`. While trying to track down flags, teams must also manage a
suite of services (see [Service Checker](#service-checker)), which are combined
with the CTF results, and reflected on the scoreboard in real-time.

Most config options for the CTF event are your standard set of tweaks for a web
server, such as host, ports, (optional) SSL cert locations, etc.

#### Users and Roles

Contestant & Administrator users are configured through the web interface.
Detailed instructions and sample data are available on the admin dashboard,
once logged in.

Users are divided up into different roles, known as teams, just as they would
be called on during the competition:

| Team       | Role                                 |
| ---------- | ------------------------------------ |
| blueteam   | Participant (Students)               |
| ctf_creator| CTF designers (Support staff)        |
| admin      | Infrastructure (Reading this README) |

* Blues can submit flags, and appear on the scoreboard
* CTF staff can view/modify challenges, and see more detailed analytics
  surrounding the challenges as the competition is running.
* Admins can see everything and modify users/reset passwords

#### Running the Web Server

To get the **cyboard server** up and running:

1. Ensure **postgres** is setup and running
2. Configure your `config.toml` based on the example
    - You can double-check your settings are picked up right with:
      `./cyboard server --config [path-to-config.toml] --dump-config`
3. Start the server:
    - `./cyboard server --config [path-to-config.toml]`
4. Browse to the welcome page:
    - By default: http://127.0.0.1:8080/

_Note on SSL:_ To quickly enable ssl using self-signed certs, you may run:
    `go run ./setup/generate_cert.go --host https://127.0.0.1 --rsa-bits 4048`


### Service Monitor

> Config File: `config.toml` -> `[service_monitor]` section

> Logs: `checks.log`

This component awards points to teams by monitoring the stability of their
infrastructure. Typically, this may be services such as a router, mail server,
e-commerce website, and file share server. In cyber defense competitions, teams
are typically trying to compete against each other, while fending off professional
red team members that represent an omnipotent, adversarial threat.

A set of service checks - scripts/programs on the filesystem - are run at
regular intervals, which determine the health of each service. The global
interval and check timeout are configured in `config.toml`. Individual services
are configured in the web ui, where you set these attributes for each:

* Name
* Description
* Start time
* Enabled/Disabled
* Total Point Value
* Script file name
* Commandline arguments
* _etc._

Any changes in the web ui will be detected by the service monitor and it will
use the new settings at the beginning of the next check interval. You can also
test your configured services from the web ui with any command arguments, to
troubleshoot from the server's standpoint and environment.

#### Writing Service Checking Scripts

The service checker - `cyboard checks` - works by running a configured script
against every blue team's IP address and checking the [exit code][exit-code]
for success, failure, or a partial success. This is a flexible strategy that we
adopted based on a suggestion to follow the paradigm of the battle-tested
[Nagios monitoring engine][nagios] (aka Naemon, Ichinga, Thruk, ...)

Points are awarded to each team are based on the check script's exit code.

The **Scoreboard** on the [Web Site](#ctf-event-and-web-server) maps exit
codes as follows:
- `Exit 0`: 'Success' (green) (only way points are awarded)
- `Exit 1`: 'Warning' (yellow)
- `Exit 2`: 'Failure' (red)
- Anything else: 'Unknown' (gray)
    - This would typically be the result of the checker timing out

The only requirement of a check is that it must accept an IP address as an
argument. Other than that and the use of exit codes, any language or binary
installed by the sysadmin to the server may be used.

In our experience, many service checks are simple shell scripts, less than
a dozen lines. On the other hand, we have also used & modified the
[Nagios Plugins programs][nagios-plugins], which you may use or gain
inspiration from.

##### Example 1
To check if a web server is up, a script may use the `curl`
command, and exit with a status of 0 if the [HTTP response is 200][httpstatus],
otherwise exit with 2.

##### Example 2
The above example may be extended further, to verify that some
text content is available on the webpage, like "Theodore Logan is the best".
If the content is present, exit 0. If the server is up, but the content is
missing or vandalized, exit 1. For all other cases, exit 2.

##### Example 3
DNS can be tested by issuing a `dig` command to a specific
name server. In general, the best way to monitor some piece of infrastructure,
is to script something that just tries to use it!

#### Running the Service Monitor

To get the service monitor running:

1. Ensure **postgres** is setup and running
2. Configure your `config.toml` based on the example
    - You can double-check your settings are picked up right with:
      `./cyboard checks --config [path-to-config.toml] --dump-config`
3. Start the server:
    - `./cyboard checks --config [path-to-config.toml]`
4. Tail the log make further changes through the web ui. They will automatically
   be reflected in the running service monitor process.

### Scheduled Event Start, End and Intermissions

> Config File: `config.toml` -> `[event]` section

The entire application is bound by the schedule in config.toml. These are times
specified with the `start`, `end`, and `breaks` options.

**It is essential that the schedule not be modified after the event begins.**
Modifying the event schedule and restarting the web server will cause teams
scores to shift around and generally upset everyone.

The web server will only display a countdown to participants until the event
starts. Admins & CTF staff can log in at any time by going to the `/login`
page, to prepare the server for the event.

Once the event begins, the CTF submissions and Service monitor will follow the
set schedule, automatically pausing during breaks or shutting down when
the event ends.


### PostgreSQL

PostgreSQL (PG) is used as a database backend for `cyboard`. To connect the app
to PG, you configure a standard [connection-string][postgres-conn] in one of
several places. In order, from lowest to highest priority:
- In the `config.toml` file:
    ```toml
    [database]
    postgres_uri = "dbname=cyboard user=cyboard host=127.0.0.1 sslmode=disable"
    # or similarly
    postgres_uri = "postgresql://localhost/cyboard?sslmode=disable"
    ```
- Environment variable: `CY_POSTGRES_URI`
- Command line parameter: `--postgres-uri "postgresql://..."`

You can connect to PostgreSQL yourself and browse tables using the standard
`psql` tool, `pgadmin` web app, or anything that can speak to Postgres. The
database tablespace faily small and easy to navigate. Make sure you are looking
in the correct schema by setting your `search_path`. To permanently add cyboard
to your search path, set it within psql:
`ALTER ROLE myuser SET search_path = "cyboard" [, other schemas ...]`

As a convenience, remember that you can set your default settings for Postgres
connections using [environment variables][postgres-env], like `$PGDATABASE`.

## Docker

Docker deployments are supported! For more info, check out the docs in
`./setup/docker/`.

-----

[LICENSE](LICENSE.txt)


<!-- Footnote Back links -->

[exit-code]: https://en.wikipedia.org/wiki/Exit_status
[dep]: https://golang.github.io/dep/
[golang]: https://golang.org/
[go-install]: https://golang.org/doc/install]
[gopath]: https://golang.org/doc/code.html#GOPATH
[httpstatus]: https://httpstatuses.com/
[nagios]: https://www.nagios.org/projects/nagios-core/
[nagios-plugins]: https://github.com/nagios-plugins/nagios-plugins
[postgres-conn]: https://www.postgresql.org/docs/10/libpq-connect.html#LIBPQ-CONNSTRING
[postgres-dl]: https://www.postgresql.org/download/
[postgres-env]: https://www.postgresql.org/docs/10/libpq-envars.html
[postgres-hba]: https://www.postgresql.org/docs/10/auth-pg-hba-conf.html#EXAMPLE-PG-HBA.CONF
[timescale-home]: https://www.timescale.com/
[timescale-install]: https://docs.timescale.com/v0.11/getting-started/installation
[toml]: https://github.com/toml-lang/toml
