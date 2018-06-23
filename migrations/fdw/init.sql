-- Config the name of the database in Mongo. The default used before was 'scorengine'.
\set MONGO_DB scorengine


CREATE SCHEMA IF NOT EXISTS mgo;
CREATE EXTENSION IF NOT EXISTS hstore WITH SCHEMA mgo;

-- Configure the wrapper.
-- NOTE: You may need to change the address/port, or the USER MAPPING if you have authorization enabled
CREATE EXTENSION IF NOT EXISTS mongo_fdw WITH SCHEMA mgo;
CREATE SERVER mgo FOREIGN DATA WRAPPER mongo_fdw OPTIONS (address '127.0.0.1', port '27017');
CREATE USER MAPPING FOR PUBLIC SERVER mgo;


-- Create the foreign tables, mapping all original BSON fields to Postgres columns

CREATE FOREIGN TABLE mgo.teams(
  _id NAME,
  "group" TEXT,
  number int,
  name TEXT,
  ip TEXT,
  hash TEXT,
  adminof TEXT
) SERVER mgo OPTIONS (database :'MONGO_DB', collection 'teams');

CREATE FOREIGN TABLE mgo.challenges(
  _id NAME,
  "group" TEXT,
  name TEXT,
  description TEXT,
  flag TEXT,
  points int
) SERVER mgo OPTIONS (database :'MONGO_DB', collection 'challenges');

CREATE FOREIGN TABLE mgo.results(
  _id NAME,
  timestamp TIMESTAMPTZ,
  "type" TEXT,
  "group" TEXT,
  teamname TEXT,
  teamnumber int,
  details TEXT,
  points int
) SERVER mgo OPTIONS (database :'MONGO_DB', collection 'results')

-- Done

