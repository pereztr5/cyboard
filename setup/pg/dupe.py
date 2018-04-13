#!/usr/bin/env python3
"""dupe.py mirrors a cyboard mongodb results collection into a postgres table,
for analysis in Grafana.

Spring 2018, by Butters

==============================================
WARNING: THIS SCRIPT WILL DELETE ALL DATA IN THE POSTGRES `cy.result` TABLE

    !!! USE WITH CAUTION !!!
==============================================

The dupe script is a fallback in case the two databases become out of sync in some
unforseen way. Eventually, the goal is to fully migrate to a database like Postgres
with richer query support.

This script was spliced together from a few smaller scripts, which is why it's full of
calls to other CLI tools, like `mongoexport`. This had the handy effect of not requiring
any external python libraries, which is great because dependency management in python is
confusing to explain to non-developers.
"""

import csv
import datetime
import gzip
import os
import shutil
import subprocess
import sys
import tempfile

print("dupe.py: Mirror scores from Cyboard's MongoDB into Postgres")
print()

if sys.version_info < (3,5):
    sys.exit("ERROR: Python v3.5 or greater is required!")

"""
Script Configuration:
(if you need password auth for Postgres, use a `pgpass` file!)
"""

mongo = {
    'db': 'scorengine',
    'collection': 'results',
    'csv-header': "type,timestamp,group,teamname,teamnumber,details,points",
}

postgres = {
    'user': 'cyboard_admin',
    'db': 'cyboard',
    'table': 'cy.result',
    'csv-header': "timestamp,teamname,teamnumber,points,type,category,challenge_name,exit_status",
    'setup-script': './cyboard_schema.sql',
}


def echo(msg):
    print()
    print(msg)
    print("========================================")

# Check for required commands and files
def require_file(cfg_item, file):
    if not os.path.exists(file):
        # This line remains commented to remember the fallen f-string, which I will never get to use because centos is forever old as dirt
        # sys.exit(f"File {cfg_item} not found: {file}")
        sys.exit("File `{}` not found: {}".format(cfg_item, file))

def require_cmd(cmd):
    if not shutil.which(cmd):
        sys.exit("Missing required command: "+cmd)

require_file('PG SQL setup script', postgres['setup-script'])
require_cmd("mongodump")
require_cmd("psql")


echo("CLEARING EXISTING POSTGRES")
# Start by clearing postgres, after a prompt
ans = input('This will drop the "{}" table. Ok? [Y/n] '.format(postgres["table"]))
if ans.lower() == 'n':
    sys.exit('Nothing was done. Exiting.')

sql_args = "psql -d {db} -U {user}".format(db=postgres['db'], user=postgres['user']).split()
sql_stmt_drop_old_table = b"DROP TABLE IF EXISTS %s;" % postgres['table'].encode()
subprocess.run(sql_args, input=sql_stmt_drop_old_table)

echo("INITIALIZING SCORES TABLE")
# Then rebuild the table schema from the sql script
with open(postgres['setup-script'], 'rt', encoding='utf-8') as sql_script:
    subprocess.run(sql_args, stdin=sql_script)


echo("COPYING DATA OUT OF MONGODB")
with tempfile.NamedTemporaryFile(mode='w+t', encoding='utf-8') as tmp_mongo, tempfile.NamedTemporaryFile(mode='w+t', encoding='utf-8') as tmp_pg:
    # Copy all data out of mongodb with the export tool
    mongo_export_args = "mongoexport -d {db} -c {coll} --type=csv --noHeaderLine --fields={header}".format(
                         db=mongo['db'], coll=mongo['collection'], header=mongo['csv-header']).split()
    subprocess.run(mongo_export_args, stdout=tmp_mongo)

    # Flushing is required or the script won't copy every document from mongo
    tmp_mongo.flush()
    os.fsync(tmp_mongo)
    tmp_mongo.seek(0)

    # Reformat the dumped mongo csv for Postgres
    reader = csv.reader(tmp_mongo)
    writer = csv.writer(tmp_pg)

    for mongo_row in reader:
        # Juggle the structure and data types around from Mongo's into the expected Postgres columns
        type, ts, group, teamname, teamnumber, details, points = mongo_row
        category = group
        # Only CTF results have challenge names
        if type == "CTF":
            challenge_name = details
            exit_status = ""
        # and only Service results have an exit status
        else:
            challenge_name = ""
            st = details.split(": ")[-1]
            exit_status = int(st) if st.isnumeric() else 3
        pg_row = [ts, teamname, teamnumber, points, type, category, challenge_name, exit_status]

        writer.writerow(pg_row)

    # Flushing is required or the script won't copy every document from mongo
    tmp_pg.flush()
    os.fsync(tmp_pg)

    echo("IMPORTING CLEANED UP DATA FROM MONGO INTO POSTGRES")

    sql_stmt_copy_csv = b"\\COPY %s FROM '%s' WITH (FORMAT csv)" % (postgres['table'].encode(), tmp_pg.name.encode())
    subprocess.run(sql_args + ["-c", sql_stmt_copy_csv])

print()
print("All set!")
