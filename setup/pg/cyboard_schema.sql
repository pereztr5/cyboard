--
-- PostgreSQL database dump
--

-- Dumped from database version 10.3
-- Dumped by pg_dump version 10.3

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: timescaledb; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS timescaledb WITH SCHEMA public;


--
-- Name: EXTENSION timescaledb; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION timescaledb IS 'Enables scalable inserts and complex queries for time-series data';


--
-- Name: cy; Type: SCHEMA; Schema: -; Owner: x
--

CREATE SCHEMA cy;


ALTER SCHEMA cy OWNER TO cyboard_admin;


--
-- Name: tablefunc; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS tablefunc WITH SCHEMA public;


--
-- Name: EXTENSION tablefunc; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION tablefunc IS 'functions that manipulate whole tables, including crosstab';



SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: result; Type: TABLE; Schema: cy; Owner: cyboard_admin
--

CREATE TABLE cy.result (
    "timestamp" timestamp without time zone NOT NULL,
    teamname text NOT NULL,
    teamnumber smallint NOT NULL,
    points integer NOT NULL,
    type text NOT NULL,
    category text NOT NULL,
    challenge_name text,
    exit_status integer
);


ALTER TABLE cy.result OWNER TO cyboard_admin;

--
-- Name: TABLE result; Type: COMMENT; Schema: cy; Owner: cyboard_admin
--

COMMENT ON TABLE cy.result IS 'Holla holla get up on dat scoreboard!!';


--
-- Name: COLUMN result.type; Type: COMMENT; Schema: cy; Owner: cyboard_admin
--

COMMENT ON COLUMN cy.result.type IS 'Either "CTF" or "Service"';


--
-- Name: COLUMN result.category; Type: COMMENT; Schema: cy; Owner: cyboard_admin
--

COMMENT ON COLUMN cy.result.category IS 'Was "group" field. Challenge group like "Wireless" or Service Check like "Router Ping"';


--
-- Name: COLUMN result.challenge_name; Type: COMMENT; Schema: cy; Owner: cyboard_admin
--

COMMENT ON COLUMN cy.result.challenge_name IS '(from "details") Only for "CTF"; Should be unique per team';


--
-- Name: COLUMN result.exit_status; Type: COMMENT; Schema: cy; Owner: cyboard_admin
--

COMMENT ON COLUMN cy.result.exit_status IS '(from "details") Exit code for Service, which looks like "Status: 0" or "Status: timed out"';


--
-- Name: result_category_index; Type: INDEX; Schema: cy; Owner: cyboard_admin
--

CREATE INDEX result_category_index ON cy.result USING btree (category);


--
-- Name: result_teamname_index; Type: INDEX; Schema: cy; Owner: cyboard_admin
--

CREATE INDEX result_teamname_index ON cy.result USING btree (teamname);


--
-- Name: result_timestamp_idx; Type: INDEX; Schema: cy; Owner: cyboard_admin
--

CREATE INDEX result_timestamp_idx ON cy.result USING btree ("timestamp" DESC);


--
-- Name: result_type_index; Type: INDEX; Schema: cy; Owner: cyboard_admin
--

CREATE INDEX result_type_index ON cy.result USING btree (type);


--
-- Name: teams_name_uindex; Type: INDEX; Schema: cy; Owner: cyboard_admin
--

CREATE UNIQUE INDEX teams_name_uindex ON cy.teams USING btree (name);


--
-- Setup Timescale Hypertable
--
SELECT create_hypertable('cy.result', 'timestamp')


--
-- Name: SCHEMA cy; Type: ACL; Schema: -; Owner: cyboard_admin
--

GRANT USAGE ON SCHEMA cy TO grafana;


--
-- Name: TABLE result; Type: ACL; Schema: cy; Owner: cyboard_admin
--

GRANT SELECT ON TABLE cy.result TO grafana;


--
-- PostgreSQL database dump complete
--

