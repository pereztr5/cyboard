# Managing TimescaleDB in Postgres

**Note:** It is VERY helpful to set postgres connection env vars, to avoid having
to supply connection string info everywhere: export `PGHOST`, `PGDATABASE`, and `PGUSER`

## Upgrading timescale and dbs with it

https://docs.timescale.com/v1.3/using-timescaledb/update-db

1. Take a data backup: `pg_dump -Fc db.dump`
2. [Install a newer timescaledb](https://docs.timescale.com/latest/getting-started/installation)
3. Restart postgres
4. Update: `psql -X -c 'ALTER EXTENSION timescaledb UPDATE;'`
5. Check with `psql ... -c '\dx timescaledb'`


## Upgrading Postgres

One of two ways:

1. Or perform a backup & restore
    * You can restore from backups of older postgres versions, and older timescale versions.
    * _Again: Make sure timescaledb is installed on the new Postgres_
    * I have found this much more reliable than option 2.

2. [pg_upgrade](https://www.postgresql.org/docs/current/pgupgrade.html)
    * `pg_upgrade -b oldbindir -B newbindir -d olddatadir -D newdatadir -O "-c timescaledb.restoring='on'"`
    * That's "bindir", as in the 'PostgreSQL executable directory'
    * Default data dirs are usually `/var/lib/pgsql/data/...`
    * _Make sure timescaledb is installed on the new Postgres_



## Backups & Restoring db dumps

Backups start with standard tools:

```sh
    pg_dump -Fc -f backup.dump cyboard
    pg_restore --list backup.dump   # Inspect a db dump. Shows schema & table info.
```

And then restoring requires a few additional steps from `psql`.
For [timescaledb v1.3](https://docs.timescale.com/v1.3/using-timescaledb/backup) and up:

```sql
    \set target cyboard
    CREATE DATABASE :target OWNER :target;
    \c :target --connect to the db where we'll perform the restore
    CREATE EXTENSION timescaledb;
    SELECT timescaledb_pre_restore();
    \! pg_restore -Fc -d :target backup.dump
    SELECT timescaledb_post_restore();
```

For [timescaledb v1.2](https://docs.timescale.com/v1.2/using-timescaledb/backup) and below:

```sql
    \set target cyboard
    CREATE DATABASE :target OWNER :target;
    ALTER DATABASE :target SET timescaledb.restoring='on';
    \c :target
    \! pg_restore -Fc -d :target backup.dump
    ALTER DATABASE :target SET timescaledb.restoring='off';
```

Backing up and restoring individual tables requires a different ceremony. See the docs.
