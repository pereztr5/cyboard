BEGIN;

CREATE TABLE IF NOT EXISTS distributed_service_check (
      LIKE service_check INCLUDING ALL
    , FOREIGN KEY (team_id) REFERENCES team (id)
    , FOREIGN KEY (service_id) REFERENCES service (id)
);

COMMIT;
