-- Copy data from Mongo's schema to Postgres

INSERT INTO team (name, role_name, hash, disabled, blueteam_ip)
   (SELECT
     name,
     (CASE WHEN "group" IN ('admin', 'blackteam') THEN 'admin'
           WHEN "group" <> 'blueteam' THEN 'ctf_creator'
           ELSE 'blueteam' END)::team_role AS role_name,
     hash::bytea,
     'false' AS disabled,
     (CASE WHEN "group" = 'blueteam' THEN split_part(ip, '.', 4)::int
           ELSE NULL END) AS ip
   FROM mgo.teams);

INSERT INTO service (name, category, description, points)
  (SELECT "group" AS name, 'unknown' AS category, '' AS description, MAX(points)
  FROM mgo.results WHERE type = 'Service' GROUP BY "group");

INSERT INTO service_check (created_at, team_id, service_id, exit_code, status)
  (SELECT
    timestamp,
    team.id AS team_id,
    service.id AS service_id,
    (CASE WHEN z.details = 'timed out' THEN 99 ELSE z.details::int END) AS exit_code,
    (CASE WHEN z.details = 'timed out' THEN 'timeout'
          WHEN z.details = '0' THEN 'pass'
          WHEN z.details = '1' THEN 'partial'
          ELSE 'fail' END)::exit_status AS status
    FROM mgo.results
        JOIN team ON mgo.results.teamname = team.name
        JOIN service ON mgo.results."group" = service.name
        , LATERAL (SELECT split_part(details, ': ', 2) AS details) AS z
    WHERE type = 'Service');


INSERT INTO challenge_category (name)
  (SELECT DISTINCT "group"
  FROM mgo.challenges);

INSERT INTO challenge (name, designer, category, body, flag, total)
  (SELECT name, "group", split_part(name, '-', 1), description, flag, points
  FROM mgo.challenges);

INSERT INTO ctf_solve (created_at, team_id, challenge_id)
   (SELECT timestamp, team.id, challenge.id
    FROM mgo.results
      JOIN team ON mgo.results.teamname = team.name
      JOIN challenge ON mgo.results.details = challenge.name);

