## v0.2019.11 (Fall 2019)

### Developers:
- Switch golang dependency management from `dep` => `go mod`
- Simplify DB management (schema, user, privs), easier to import DB backups.
- Update Docker files & docs to more easily stand up just PG + TimescaleDB

### Discord:
- Add sample support scripts, which use the web api as any client could (such as bots)
- Add python script to activate challenge, then announce on Discord
- Add python script to announce the first team to solve each challenge, as that happens

### Web site:
- Blues: Add cnyhackathon-specific tips to dashboard
- CTF: Improve handling of special chars in uploaded filenames (can now a few chars like '!' and '%')
- CTF: Add API to get ctf solves (team+chal+timestamp), ordered+filtered by time
- CTF: Add API to get challenge by name
- CTF: Add API to 'activate' challenge
- CTF: Allow CTF challenges to be deleted after being solved (useful during event staging)
- CTF+Admin: Add web page/api to tail log server log files to the browser
- Admin: Show configured event breaks on Services cfg page
- Admin: Add API to get event configuration as JSON
- Admin: When adding new Services, the Service start time is pre-populated at the event start time (fixes one of the most tedious manual parts of the event setup)
- Other minor Bug fixes and UI fixes


## v0.2019.09 (NSA Hackathon 2019)

### General:

- Update Docker deployment to drop Mongo and support Postgres + TimescaleDB (v10.6 + v0.11)
- Rewrite README.md to feature Postgres setup, tips&tricks, Web UI configuration added last year, and a lot more documentation.


## v0.2018.11 (Fall 2018)

*Complete rewrite over the 2018 summer/fall*

### Migrate DB to Postgres:
- Significant performance improvements
- Join operations enable normalized data (easier management, able to rename teams, services, etc)
- Transactions semantics prevent races (double-submit of challenge solve)
- SQL enables advanced analytics that weren't possible with MongoDB

### Web server rewrite:
- Routes: Normalized web router for both pages and API endpoints (consistency around CRUD ops)
- Routes: Add a health check endpoint, /ping 🦆
- Routes: Add APIs for managing teams, services, ctf, files for ctf, and 'other' points
- Routes: Add admin routes to get all active blueteams, and a team by name
- Middleware: Restrict Blueteam actions to event times - time-aware routes
- Middleware: Add optional compression middleware
- Middleware: Add optional rate limiting on /login and blueteam flag guesses
- Auth: Persist session signing secret to DB (stay logged in after server restart)
- Auth: Switch from gorilla/sessions to alexedwards/scs
- Auth: Cookies last 1 week, option to share over http
- Auth: Protect cookies with SameSite "Lax" (CSRF protection for modern browsers)
- Ctf: Separate 'public' and 'hidden' challenges
- Ctf: Add challenge file management (that's what CMS stands for, yes?)
- General: Respect config ip/interface setting; Cleanup http->https redirect
- General: Make server shutdown gracefully
- General: Add server connection timeouts & secure tls settings
- General: Add robots.txt
- General: Add a site favicon (looks like a little graph)

### Service Monitor/Checks:
- Postgres enables service updates mid-event; Listen/notify catch changes automatically
- Add algo to easily pre-calculate service points to award per check
- Automate start & stop of service checker; no more manual actions required
- Automated starting of individual services (at each service's `StartsAt` time)
- Add more flexibility to arg vars (new options, expand like bash args)
- Simplify service config with base_ip_prefix

### Website:
- General: Update to bootstrap 4 & jquery 3
- General: Serve all dependencies locally
- General: Miscellaneous UI tweaks (page titles, vids)
- General: Rewrote every webpage (home, scoreboard, ctf, and admin event management)
- Public: Combine scoreboard and services pages into one
- Public: Services display automatically adds new services when their checks begin
- Public: Services Legend, many responsive-ness tweaks, up contrast/'pop'
- Public: Challenge descriptions & files now shown right next to flag submission box
- Public: Group public challenges by category
- Public: Add countdown timer; restrict access until event begins
- Admin: Add admin Services management page (configure checks, view event times, scripts)
- Admin: Add admin CTF dashboard (shows stats: first solver, each team's solves, timestamps)
- Admin: Add admin page for awarding bonus/other points (shows all bonuses awarded so far)
- Admin: Display total alloted points for ctf & services

### General:
- Switch golang dependency management from `glide` => `dep`
- Combine app command flags, swap mongo refs for postgres
- Combine two config files into one shared config
- Validate config file on startup
- Added config for "scheduled breaks", fully integrated into web server and service monitor
- Move web dirs tmpl/ and static/ together under ui/
- Many code quality improvements
- Added many new tests; must be run in serial w/ live Postgres DB (go test -p <...>)

## v0.2018.04 (Spring 2018)

- BREAKING: Dropped support for Go 1.8
- Add jitter to the service checking interval
- Add support for exploring event score data in Grafana, via Postgres DB

## v0.2017.11 (Fall 2017)

- Display scores per category for each team. This allows organizers and contestants to know who is doing better in CTF vs. Infrastructure (#27)
- Add component to configure the CTF from the website, via CSV text upload (#30)
- Add API endpoint & component to give out Bonus Points 🏆 as well as point deductions (#29)
- Huge performance improvements (#28)
- Encapsulate app configuration within the code base (#21)
- Significantly expand project documentation (#22, #29)
- Integrate Travis-CI build support, add `docker` & `docker-compose` support (#22)
- Increased test coverage (from 0% to 12%. _Progress!_)

## Older Releases

- In the beginning, there was things and stuff...
