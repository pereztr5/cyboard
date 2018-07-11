BEGIN;

-- Comments provided to help remind myself later what I was thinking with this.


-- SQL script made to work with: https://github.com/golang-migrate/migrate/
-- Though, `golang-migrate` has some quirks that may not make it the best choice.
-- See: https://github.com/golang-migrate/migrate/issues/34 (does not play well with postgres schemas)
--      https://github.com/mattes/migrate/issues/13
--      https://github.com/mattes/migrate/issues/274


-- First: Schema, application user w/ priviledges, and extensions
CREATE SCHEMA IF NOT EXISTS cyboard;

----------------
-- Must create role separately.
--
-- TODO: Add instructions to readme
-- $ createuser --login -h localhost -U postgres cybot
----------------
-- CREATE ROLE cybot LOGIN;

ALTER ROLE cybot SET search_path = cyboard;

GRANT USAGE ON SCHEMA cyboard TO cybot;
ALTER DEFAULT PRIVILEGES IN SCHEMA cyboard
    GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE
    ON TABLES TO cybot;

SET search_path = cyboard;

CREATE EXTENSION IF NOT EXISTS timescaledb ;  -- Better time-series data support in Postgres
                                              -- https://github.com/timescale/timescaledb/
CREATE EXTENSION IF NOT EXISTS moddatetime ; -- Provides functions for tracking modification time
CREATE EXTENSION IF NOT EXISTS tablefunc ; -- Provides functions for crosstab (pivot tables)

----------------
-- Configuration
----------------
CREATE TABLE config (
      key   TEXT NOT NULL UNIQUE
    , value TEXT NOT NULL
);

----------------
-- User Accounts
----------------

/* team_role */
CREATE TYPE team_role AS ENUM (
      'admin'
    , 'ctf_creator'
    , 'blueteam' -- contestants, students
);

/* The 'users' table. It was `team` before, which is fine. It's understandable and short.

'role_name' represents a group of users, of which many teams may be a part of,
and their permission will be controlled by a separate, yet-to-be-designed table.
*/
CREATE TABLE team (
      id           INT       PRIMARY KEY GENERATED ALWAYS AS IDENTITY
    , name         TEXT      NOT NULL UNIQUE
    , role_name    team_role NOT NULL
    , hash         BYTEA     NOT NULL
    , disabled     BOOL      NOT NULL DEFAULT false

    , blueteam_ip  SMALLINT   NULL

    /*
    This is a two-way check. Only contestants (blueteam) must have an ip octect.
    No other team_role (staff, ctf designers) need an ip, so they *can't* have one,
    because it was awkward when it was like that before.

    Instead of making a whole enhanced entity relationship table model for blueteam's attributes and
    trying to enforce the constraint across tables (cludgy!), this one attribute is enforced here.
    */
    , CONSTRAINT only_blueteam_needs_ip
        CHECK ((role_name = 'blueteam') != (blueteam_ip IS NULL))
);

-- The IP for the blueteams must be unique
CREATE UNIQUE INDEX blueteam_ip_uni_idx
    ON team (blueteam_ip)
    WHERE role_name = 'blueteam';


----------------
-- CTF Challenge
----------------

-- Challenges are designed by different volunteers, who place them into categories (Reversing, Web, etc.)
CREATE TABLE challenge_category (name TEXT PRIMARY KEY);

/*
Challenges are solved by contestants entering the exact magic string, held in `flag`.

A description of the challenge is saved in `body`, which can be displayed to contestants.
I'm thinking this will be markdown or html, in which case it would be best to save it as a file,
which would mean updating the table schema here.
*/
CREATE TABLE challenge (
      id        INT     PRIMARY KEY GENERATED ALWAYS AS IDENTITY
    , name      TEXT    NOT NULL UNIQUE
    , category  TEXT    NOT NULL DEFAULT ''
    , flag      TEXT    NOT NULL UNIQUE
    , total     REAL    NOT NULL DEFAULT 0.0
    , body      TEXT    NOT NULL DEFAULT ''
    , hidden    BOOL    NOT NULL DEFAULT FALSE

    , designer_category TEXT NOT NULL REFERENCES challenge_category(name) ON DELETE CASCADE ON UPDATE CASCADE

    , created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
    , modified_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER mdt_challenge
    BEFORE UPDATE ON challenge
    FOR EACH ROW
    EXECUTE PROCEDURE moddatetime (modified_at);


/* Files served with a given ctf challenge. E.G. `crackme` binaries, encrypted messages, etc. */
CREATE TABLE challenge_file (
      id           INT  PRIMARY KEY GENERATED ALWAYS AS IDENTITY
    , challenge_id INT  NOT NULL REFERENCES challenge(id) ON DELETE CASCADE
    , filename     TEXT NOT NULL
    , description  TEXT NOT NULL DEFAULT ''
);

--------------------
-- Monitored Service
--------------------

/*
A service is checked periodically for uptime / 'correctness', for each team.
The check is run as a command `script`, which is a binary/script on the central monitoring server.
Management of these scripts is all done on the server itself, not the web ui (yet?).

Service checking is staggered.
A service will only first start being monitored once the time `starts_at` passes.
If just one service needs to be disabled after starting, there's a toggle field for that.


In this table, `total_points` is the expected max for this service across the event.
Meanwhile, `points` represents the actual amount awarded per passing check, per team.

On first run of the monitoring script, if the `points` field is null, it will be set based on
the `total_points` field, divided across the expected amount of check attempts for that service.
See the comment above the `service_check` table for further details.
*/
CREATE TABLE service (
      id           INT    PRIMARY KEY GENERATED ALWAYS AS IDENTITY
    , name         TEXT   NOT NULL UNIQUE
    , category     TEXT   NOT NULL
    , description  TEXT   NOT NULL -- How to score
    , total_points REAL   NOT NULL DEFAULT 0.0
    , points       REAL   NULL
    , script       TEXT   NOT NULL DEFAULT ''
    , args         TEXT[] NOT NULL DEFAULT '{}'
    , disabled     BOOL   NOT NULL DEFAULT true

    , starts_at   TIMESTAMPTZ NOT NULL DEFAULT '-infinity'::TIMESTAMPTZ
    , created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
    , modified_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER mdt_service
    BEFORE UPDATE ON service
    FOR EACH ROW
    EXECUTE PROCEDURE moddatetime (modified_at);


-----------------
-- Scoring Tables
-----------------

CREATE TYPE exit_status AS ENUM ('pass', 'fail', 'partial', 'timeout');

/*
service_check is an individual run of the service monitor against a team's infrastructure.

This is by far the largest table in the application (50,000+ rows; which isn't really that big, but still).
Indexes should be chosen with care, as this is one of the only places it will actually matter!

I've considered a roll-up table that aggregates this data every 5 minutes or so, to keep
the data lighter. It could also be a materialized view, but I'm not clear on the restrictions
they have just yet.


Scoring itself is somewhat complex, because the mixed event style means that an individual check can't
simply be worth, say, 5 points, because the total score the service can generate has to be
proportional to the scores the CTF challenges can generate.

As an example:
If we set `points` to 100.0, and at the top level config
set check interval set to 15s, and an event spanning 8 hours event w/ a 1.25 hour break for lunch,
each passing check would be worth
8h - 1.25h = 6.75 hrs; 24300 seconds
24300s / 15s = 1620 checks
1000.0 pts / 1620 chks = 0.617~ points per check

The amount per check will be calculated and saved on at first run.

This method has drawbacks in case the event is delayed, or runs late
(both of which have happened every single time), which will skew the total
amount of points generated by a service, since it is primarily based on
how long the service was checked for.
*/
CREATE TABLE service_check (
      created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
    , team_id     INT          NOT NULL REFERENCES team(id)
    , service_id  INT          NOT NULL REFERENCES service(id)
    , status      exit_status  NOT NULL -- determines points awarded, and status display in web ui
    , exit_code   SMALLINT     NOT NULL -- actual exit code, for debugging
);

-- CREATE INDEX service_check_fkey_idx_team    ON service_check (team_id);
-- CREATE INDEX service_check_fkey_idx_service ON service_check (service_id);
-- CREATE INDEX service_check_idx_status ON service_check (status);

-- ctf_solve is a timestamp of when a team solved a challenge
CREATE TABLE ctf_solve  (
      created_at   TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
    , team_id      INT          NOT NULL REFERENCES team(id)
    , challenge_id INT          NOT NULL REFERENCES challenge(id)

    /* , UNIQUE (team_id, challenge_id) */
    /*
    In timescaledb, unique constraints must include the
    timestamp field, which is _not_ what we want here.
    We must enforce the uniqueness check at the application level,
    to prevent a team from scoring the same flag repeatedly.
    See: https://github.com/timescale/timescaledb/issues/488
    */
);

-- CREATE INDEX ctf_solve_fkey_idx_team      ON ctf_solve (team_id);
-- CREATE INDEX ctf_solve_fkey_idx_challenge ON ctf_solve (challenge_id);

-- other_points is for bonus points, deductions for misbehavior, etc.
CREATE TABLE other_points (
      created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
    , team_id     INT          NOT NULL REFERENCES team(id)
    , points      REAL         NOT NULL
    , reason      TEXT         NOT NULL DEFAULT ''
);

-- Activate timescaledb extension on the scoring tables
SELECT create_hypertable('service_check', 'created_at');
SELECT create_hypertable('ctf_solve',     'created_at');
SELECT create_hypertable('other_points',  'created_at');


-- Create views for scoring

CREATE VIEW blueteam (id, name, blueteam_ip)
    AS SELECT team.id, team.name, blueteam_ip
    FROM cyboard.team
    WHERE team.role_name = 'blueteam'
      AND team.disabled = false;

CREATE VIEW cyboard.service_score (team_id, points)
    AS SELECT team.id, COALESCE(sum(service.points), 0)
    FROM cyboard.blueteam AS team
        LEFT JOIN service_check AS sc ON team.id = sc.team_id AND sc.status = 'pass'
        LEFT JOIN service ON sc.service_id = service.id
    GROUP BY team.id;

CREATE VIEW cyboard.ctf_score (team_id, points)
    AS SELECT team.id, COALESCE(sum(ch.total), 0)
    FROM cyboard.blueteam AS team
        LEFT JOIN ctf_solve ON team.id = ctf_solve.team_id
        LEFT JOIN challenge AS ch ON ctf_solve.challenge_id = ch.id
    GROUP BY team.id;

CREATE VIEW cyboard.other_score (team_id, points)
    AS SELECT team.id, COALESCE(sum(o.points), 0)
    FROM cyboard.blueteam AS team
        LEFT JOIN other_points AS o ON team.id = o.team_id
    GROUP BY team.id;

COMMIT;