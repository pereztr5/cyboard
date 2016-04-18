mongoimport --db scorengine --collection challenges --drop --file challenges.json --jsonArray
mongoimport --db scorengine --collection teams --drop --file teams.json --jsonArray
mongoimport --db scorengine --collection results --drop --file results.json --jsonArray
