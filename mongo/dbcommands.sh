mongoimport --db scorengine --collection services --drop --file serviceChecks.json --jsonArray
mongoimport --db scorengine --collection flags --drop --file testFlags.json --jsonArray
mongoimport --db scorengine --collection teams --drop --file teams.json --jsonArray
