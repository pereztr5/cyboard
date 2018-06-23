FDW scripts
-----------

This dir contains sql scripts for porting old data dumps stored in mongo to postgres.

The goal is to use live data from old events to test the new postgres schema.
Instead of generating fuzzy data that doesn't match reality, we can use real data!

Using the postgres feature [Foreign data wrappers](https://www.postgresql.org/docs/10/static/ddl-foreign-data.html),
an existing mongo database can be queried from postgres, doing all the typical SQL stuff.
So instead of dumping mongo collections, writing ingestion scripts in bash/awk/perl/python,
and importing those into postgres, FDWs let us stay entirely within the realm of SQL.

Installing the wrapper
----------------------

Project: https://github.com/EnterpriseDB/mongo_fdw

NOTE: Mongo 3.4 or below is _REQUIRED_. Initially using Mongo 3.6 (latest) caused
      Postgres crashes on certain queries.

The readme for the `mongo_fdw` project is a little sporadic, and it assumes you
would want to build the underlying driver along with itself. The mongo-c-driver
appears to have solid maintenance in several OS package managers, so I opted to
install the dependencies that way.

The install process on Arch linux resembled the following:

```bash
# Install the C language mongo driver and json lib
sudo pacman -S mongo-c-driver json-c
# Install build dependencies
sudo pacman -S automake make gcc pkg-config

# Clone
git clone https://github.com/EnterpriseDB/mongo_fdw
cd mongo_fdw
# Optionally checkout the exact commit I used:
#git checkout 450e16537ced8266f0dedc9dd9a5669ed6737b07

# Setup, replacing the Makefile my modified one (next to this README.md)
cp -f <path_to_this_folder>/Makefile.arch  Makefile
touch config.h

# Build & install mongo_fdw
make
sudo make install

# Update postgres database configs to load the extension
vim /var/lib/postgres/data/postgresql.conf
# Find the "shared_preload_libraries" settings, and add `mongo_fdw`:
#shared_preload_libraries = 'pg_stat_statements, timescaledb, mongo_fdw'

# Restart postgres service (sometimes it might be postgresql-10, or whichever version)
sudo systemctl restart postgresql
# Check for startup errors, could indicate the library didn't link correctly
systemctl status postgresql
```

If there are problems with building or loading the library, try checking the
issues on the projects github. If you have to dig into the Makefiles, keep this
doc handy: https://www.postgresql.org/docs/10/static/extend-pgxs.html

Connect with Mongo from Postgres
--------------------------------

Once the fdw is installed, the sql scripts in this folder can clone from
the Spring 2018 mongodump.

NOTE: It's a good idea to export connection details as environment variables,
      otherwise all of the follow commands will need to be updated with flags
      for your host, user, port, etc.
      See: https://www.postgresql.org/docs/current/static/libpq-envars.html
      and: https://www.postgresql.org/docs/current/static/libpq-pgpass.html

```bash
# Setup postgres with the initial schema, unrelated to the foreign mongo tables.
# You may have already done this as part of setting up cyboard.
#psql -f ../001_initialize_schema.up.sq

# Initialize the mongo_fdw extension, and create tables w/ user mappings
psql -f ./init.sql

# Now you can query Mongo from Postgres!
# [local]:5432> SELECT * FROM mgo.teams;

# The mongo collections are available as foreign data tables under the `mgo` schema.
# TABLES: `mgo.challenges`, `mgo.teams`, and `mgo.results`

# Transfer the data out of Mongo into Postgres, adapting to the new schemas
psql -f ./populate.sql

# Now all the data from Mongo is natively available in Postgres, with typed columns and everything.
# [local]:5432> SELECT * FROM team;

# Optionally, remove all the foreign data tables and mongo connections if you
# don't need them anymore
psql -f ./cleanup.sql
```

Making Postgres Backups
-----------------------

Finally, you could save the database w/o the mongo tables, so that you don't need to keep mongo around:

```bash
# Make a compressed backup
pg_dump -f cyboard.dump -Fc

# Develop, make changes, realize you need to roll back to the backup...

# Drop the application's tables
psql -c 'DROP SCHEMA cyboard CASCADE'

# Restore
pg_restore -d cyboard --disable-triggers cyboard.dump
```


Everything should be all set. Happy developing!

