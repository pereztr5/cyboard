# Cyboard

[![Build Status](https://travis-ci.org/pereztr5/cyboard.svg?branch=master)][travis]

**Cyboard** is a modular scoring engine for cyber defense competitions.

<!-- Generated with https://github.com/jonschlinkert/markdown-toc -->
<!-- toc -->

- [Background](#background)
- [Features](#features)
- [Building](#building)
- [Administration](#administration)
  * [CTF Event and Web Server](#ctf-event-and-web-server)
    + [Public Challenges](#public-challenges)
    + [Users and Roles](#users-and-roles)
    + [Running the Web Server](#running-the-web-server)
  * [Service Checker](#service-checker)
    + [Writing Service Checking Scripts](#writing-service-checking-scripts)
    + [Scheduled Event End and Intermissions](#scheduled-event-end-and-intermissions)
    + [Running the Service Checker](#running-the-service-checker)
  * [MongoDB](#mongodb)
- [API](#api)
  * [Endpoints](#endpoints)
- [Docker](#docker)
- [Contributors](#contributors)

<!-- tocstop -->

## Background

[Tony](https://github.com/pereztr5 "Tony Perez") started developing Cyboard
in 2016 as his senior project at SUNY Polytechnic, with the goal to use it
for the [CNY Hackathon](https://www.cnyhackathon.org "CNY Hackathon Home").

CNY Hackathon is a joint cyber security defense & CTF event for intercollegiate
students of the Central New York region. The event is hosted bi-annually to
100+ contestants. Despite the name, it shares no similarities with a
[programming hackathon](https://en.wikipedia.org/wiki/Hackathon).

## Features

- Web Application Server
    - CTF event submission & display
    - Scoreboard actively maintained via WebSockets
    - JSON-based HTTP API (REST-like)
    - User & Team authentication
    - Specific functionality for blue teams, red teams, and black teams
- Service Checker
    - Scores contestants' infrastructure at regular intervals
    - Checks are any script/program, language agnositc
- Each piece is integrated through a MongoDB back end


## Building

Cyboard is written in [Golang][golang], and has been tested on recent versions of MacOS,
FreeBSD, CentOS, Arch, and Ubuntu. To build `cyboard`:

1. Install Golang
    * The simplest method is to use your system's package manager
      to download the `golang` package - e.g. `yum install golang`
    * Alternatively Download & Install [Go v1.8+][go-install]
2. _Optional_: Go demands all code be located in one central folder,
    which you may configure before proceeding: [Guide to GOPATH][gopath]
2. Install [Glide][glide], which manages Go dependencies
    * The recommended way: `curl https://glide.sh/get | sh`
3. Clone this source repo into `$GOPATH/src/github.com/pereztr5/cyboard/`,
    either through:
    * `go get github.com/pereztr5/cyboard`
    * or directly via Github
4. From the source root, run:
    ``` sh
    glide install  # Fetches & builds dependencies
    go build       # Build the project
    go test ./...  # Optional: Runs unit tests (Local MongoDB install required)
    ```
5. The result generated is a single, runnable binary, `cyboard`!

To run `cyboard`, you must have a [Mongo Database][mongo-install] the app can
connect to. [Read the configuration section](#mongodb) more details.

You can install the binary itself anywhere, with the only caveat that
the `static/` and `tmpl/` folders from this source repo must be in the
current working directory.


## Administration

Cyboard is designed as two separate components, that are kicked off with
the `cyboard` command:

1. `cyboard server`: Web Application & CTF Server
2. `cyboard checks`: Service Checker / Monitor

The reasoning behind having separate services like this has been that each
piece can be distributed onto different machines, restarted independently
if one were to kick the bucket, and developed in tandem.

Cyboard's settings are managed through a mix of config files, web gui options,
and elbow grease.

The Config files are written in [TOML][toml], which is like a souped up INI
format. Config files may be placed in the current working dir, or in
`$HOME/.cyboard/*.toml`.

### CTF Event and Web Server

> Config file: `config.toml`

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
server, such as host, ports, SSL cert locations, etc.

#### Public Challenges

``` toml
# Snippet from config.toml
special_challenges = ["Wireless", "Reverse Engineering"]
```

By default, all information about every flag is hidden from the participants.
This way, the CTF turns into a real treasure hunt. Using the `special_challeges`
option, as described above, will enable groups of challenges to be made public,
which reveals their name and description, giving the contestants a better hint
as to what they're searching for.

#### Users and Roles

Contestant & Administrator users are configured through the web interface.
Detailed instructions and sample data are available on the admin dashboard,
once logged in.

Users are divided up into different roles, known as teams, just as they would
be called on during the competition:

| Team      | Role                                    |
| --------- | --------------------------------------- |
| blueteam  | Participant (Students)                  |
| redteam   | Blue's Adversary (Hackers!)             |
| blackteam | Infrastructure (Reading this README)    |
| admin     | Configures users                        |
| whiteteam | CTF designers and all around nice folks |

* Blues can submit flags, and appear on the scoreboard
* Select white and reds who are marked as the `AdminOf` a particular CTF group
  will be able to configure flags in that group.
* Black and Admins can configure & see information about all CTF Challenges
    * Flags are considered sensitive info, so be careful!

#### Running the Web Server

To get the **cyboard server** up and running:

1. Ensure [**mongodb**](#mongodb) is installed and active
2. Configure **_config.toml_**
3. Start the server:
    - `./cyboard server --config [path-to-config.toml]`
4. Browse to the welcome page:
    - By default: http://127.0.0.1:8080/

_Note on SSL:_ To quickly enable ssl using self-signed certs, you may run:
    `go run ./setup/generate_cert.go --host https://127.0.0.1 --rsa-bits 4048`



### Service Checker

> Config File: checks.toml

> Logs: `checks.log`

This component awards points to teams by monitoring the stability of their
infrastructure. Typically, this may be services such as a router, mail server,
e-commerce website, and FTP server. In cyber defense competitions, teams are
typically trying to compete against each other, while fending off professional
red team members that represent an omnipotent, adversarial threat.

A set of checks - scripts/programs on the filesystem - are run at regular
intervals, which determine the health of each service. A single check's config
will resembled the following:

``` toml
# Snippet from checks.toml
[[checks]]
check_name = "web"
filename = "check_http"
args = "-I IP -t 5"
# Exit 0 = 10 points | exit 1 = 6 | exit 2 = 0
# See explanation in "Writing" section
points = [ 10, 6, 0 ]
# Toggles whether to run a check or not
disable = false
```

The `checks.toml` config file may be updated and the Service Checker will
**automatically reload**, so you can update the amount of points awarded on
the fly, or disable a check.

#### Writing Service Checking Scripts

The service checker - `cyboard checks` - works by running a configured script
against every blue team's IP address and checking the [exit code][exit-code]
for success, failure, or a partial success. This is a flexible strategy that we
adopted based on a suggestion to follow a simplified version of the
[Nagios monitoring engine][nagios].

Points are awarded to each team are based on the check script's exit code.
The exact amount of points is configured with the check, as an array -
`points` - in `checks.toml`. The exit code is used as an index into the
`points` array. If a script's exit code does not fit in the array bounds,
it is worth 0.

The **Scoreboard** on the [Web Site](#ctf-event-and-web-server) maps exit
codes as follows:
- `Exit 0`: 'Success' (green)
- `Exit 1`: 'Warning' (yellow)
- `Exit 2`: 'Failure' (red)
- Anything else: 'Unknown' (gray)
    - This would typically be the result of the checker timing out

The only requirement of a check is that it must accept an IP address as an
argument. Other than that and the use of exit codes, any language installed on
the server may be used, though Cyboard will generally refer to these as
"_scripts_".

**Example 1**:
To check if a web server is up, a script may use the `curl`
command, and exit with a status of 0 if the [HTTP response is 200][httpstatus],
otherwise exit with 2.

**Example 2**:
The above example may be extended further, to verify that some
text content is available on the webpage, like "Theodore Logan is the best".
If the content is present, exit 0. If the server is up, but the content is
missing or vandalized, exit 1. For all other cases, exit 2.

**Example 3**:
DNS can be tested by issuing a `dig` command to a specific
name server. In general, the best way to monitor some piece of infrastructure,
is to script something that just tries to use it!

In our experience, many scripts are simple shell scripts, less than a dozen
lines. On the other hand, we have also used & modified the
[Nagios Plugins programs][nagios-plugins], which you may use or gain
inspiration from.

#### Scheduled Event End and Intermissions

The service checker will automatically stop checking and shutdown at the
date & time configured by the `event_end_time` setting in the config file.
Temporarily breaks may be issued by setting the `on_break` option to `true`
in the config file.

#### Running the Service Checker

To get the **cyboard checks** monitor running:

1. Ensure [**mongodb**](#mongodb) is installed and active
2. Configure **_checks.toml_**
    - Make sure your `checks_dir` points to a location where the service
      monitoring scripts can be found.
    - For each of your `[[checks]]`, there should be a file in `checks_dir`
3. Start the server:
    - `./cyboard checks --config [path-to-checks.toml]`
4. Tail the log and update `checks.toml` as needed - the running checker process
    will automatically reload with new settings.


### MongoDB

MongoDB is a tire fire of a database that `cyboard` relies on. Mongo can be
set up pretty easily, and there are [plenty of docs][mongodb-docs] to help
with this process. Just be sure to install MongoDB v3.0 or greater.

For connecting with MongoDB, there are multiple ways to
[configure the MongoDB URI][mongo-uri] (Last location found will be used):
- In the config files, both `checks.toml` and `config.toml`:
    ``` toml
    [database]
    uri = "mongodb://127.0.0.1"
    ```
- Environment variable: `MONGODB_URI`
- Command line parameter: `--mongodb_uri "mongodb://127.0.0.1"`

There are a few samples of the data models available in
`./setup/mongo_samples/`, to give you a better idea of what is stored and
processed, as well as an unofficial beginner's guide to MongoDB.

If you believe you must manually edit the data in Mongo, please be careful!
Mongo's implicit data types can cause confusing problems. E.g. All numbers are
`double` values by default, when inserted or updated with the `mongo` shell.

## API

For scripted tasks or bots, Cyboard has a growing selection of API endpoints
to work with.

**Note**: The API is due for substantial changes in the future. Be advised.

You will need to be authenticated to call most endpoints. This can be
achieved a couple of ways:
- From a browser session, after having logged in manually. The cookie is
    available in the web developer tools, under the "cyboard" key name.
    - In Firefox: Hit `F12`, go to "Storage" -> "Cookies" -> [domain of cyboard]
    - In Chrome: Hit `F12`, go to "Application" -> "Cookies" -> [domain of cyboard]
    - Copy the cookie, and include it in your script or tool with each request
        to `cyboard` from then on.
- Alternatively, if your tools or interfaces support persistent session storage,
    you may log in with the `/login` POST endpoint and go about your way.

<!-- TODO: Swagger docs would be much better -->

### Endpoints

#### Function: Login
+ Endpoint: `/login`
+ Role: &lt;any&gt;
+ Methods: POST
+ Request:
    * Headers: ContentType: form
    * Body:
        - `teamname` string
        - `password` string
+ Response: text/plain
    * On success: A cookie under the key `cyboard` is saved to
        the browser. Finally, issues an HTTP redirect to `/team/dashboard`.
    * On failure: Issues HTTP redirect to `/login`

#### Function: Logout
+ Endpoint: `/logout`
+ Role: &lt;any&gt;
+ Methods: GET
+ Request: &lt;ignored&gt;
+ Response: text/plain
    * On success: The logged in user's cookie under the key `cyboard`
        will be wiped. Then, HTTP redirect to `/login`.
    * On failure: If not logged in, issues HTTP redirect to `/login`

#### Function: Submit a flag guess
+ Endpoint: `/challenges/verify`
+ Role: "blueteam"
+ Methods: POST
+ Request:
    * Headers: ContentType: form
    * Body:
        - `flag` string
        - `challenge` string
+ Response: text/plain
    * On success: Adds the flag's value to the submitting team's score,
        and writes a '0', indicating the flag was captured.
    * On failure:
        - '1': if the flag was already captured by the submitting team
        - '2': if the guessed flag was incorrect.
        - Other HTTP Status Code: Bad request, server error, etc.

#### Function: List CTF challenges
+ Endpoint: `/ctf/flags`
+ Role: "admin", "blackteam", or designated CTF group owner
+ Methods: GET
+ Request: &lt;ignored&gt;
+ Response: application/json
    * On success: Returns everything about each challenge flags that
        the requesting user has access to view.
    * On failure: Standard HTTP error code.

#### Function: List all teams
+ Endpoint: `/admin/teams`
+ Role: "admin"
+ Methods: GET
+ Request: &lt;ignored&gt;
+ Response: application/json
    * On success: Returns everything but password hashes, for every user.
    * On failure: Standard HTTP error code.

#### Function: Add or Subtract Points from score
+ Endpoint: `/black/team/bonus`
+ Role: "blackteam" or "admin"
+ Methods: POST
+ Request:
    * Headers: ContentType: json
    * Body:
        - `teams` array of strings
        - `points` integer
            + Can be positive or negative
        - `details` string
            + Provide a reason why points are being given or taken away
+ Response: text/plain
    * On success: Each teams score is updated accordingly.
    * On failure: Standard HTTP error code (4XX or 5XX) and plain-text
        reason for the error.

### Examples using [httpie](https://httpie.org/ "aitch-tee-tee-pie")

`httpie` is a handy tool that can be used for scripting API work. Here's
a demonstration, using it to poke at Cyboard:

``` bash
$ http --session cyboard-demo --form POST https://localhost:8081/login teamname=admin password=p
HTTP/1.1 302 Found
Content-Length: 0
Content-Type: text/plain; charset=utf-8
Date: Sat, 28 Oct 2017 16:36:45 GMT
Location: /team/dashboard
Set-Cookie: cyboard=<snip>; Expires=Sat, 28 Oct 2017 17:36:45 GMT; Max-Age=3600; HttpOnly

# `http` saves the authenticated cookie in the '--session', making further requests simple
$ http --session cyboard-demo --json POST https://localhost:8081/black/team/bonus teams:='["team1"]' points:=42 details="You deserve it"
HTTP/1.1 200 OK
Content-Length: 0
Content-Type: text/plain; charset=utf-8
Date: Sat, 28 Oct 2017 16:38:31 GMT

$ http --session cyboard-demo GET https://localhost:8081/admin/teams
HTTP/1.1 200 OK
Content-Length: 1070
Content-Type: application/json; charset=UTF-8
Date: Sat, 28 Oct 2017 16:41:56 GMT

[ ... ]
# Combine with the powerful json parser utility `jq` to splice and select fields!
# The following returns a newline-separated list of all challenge names
$ http --session cyboard-demo GET https://localhost:8081/ctf/flags | jq -cr '.[] | .name'
crypto-1
crypto-2
programming-1
[ ... ]
$
```

## Docker

Docker deployments are supported! For more info, check out the docs in
`./setup/docker/`.


## Contributors

Cyboard is maintained by [Butters](https://github.com/tbutts "Tyler Butters") and
[Tony](https://github.com/pereztr5 "Tony Perez").

If you have any feedback or suggestions, get in contact or
[create an issue!](https://github.com/pereztr5/cyboard/issues)

-----

[LICENSE](LICENSE.txt)


<!-- Footnote Back links -->

[exit-code]: https://en.wikipedia.org/wiki/Exit_status
[glide]: https://github.com/Masterminds/glide
[golang]: https://golang.org/
[go-install]: https://golang.org/doc/install]
[gopath]: https://golang.org/doc/code.html#GOPATH
[httpstatus]: https://httpstatuses.com/
[mongo-docs]: https://docs.mongodb.com/getting-started/shell/introduction/
[mongo-install]: https://docs.mongodb.com/manual/installation/#mongodb-community-edition
[mongo-uri]: https://docs.mongodb.com/manual/reference/connection-string/
[nagios]: https://www.nagios.org/projects/nagios-core/
[nagios-plugins]: https://github.com/nagios-plugins/nagios-plugins
[travis]: https://travis-ci.org/pereztr5/cyboard
[toml]: https://github.com/toml-lang/toml
