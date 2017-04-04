## How to use MongoDB

Collected notes on the Mongo Database

### Basics
[BSON Docs](https://docs.mongodb.org/manual/reference/bson-types/#bson-types-comparison-order)  
[Update Operator Docs](https://docs.mongodb.org/manual/reference/operator/update/)  
[Query & Projection Operator Docs](https://docs.mongodb.org/manual/reference/operator/query/)  

```
show collections
```

### Quieries

Find all in collection `flags`  
```
db.flags.find()
```

Find and match  
```
db.flags.find({ "name": "Web" })
```

Query with multiple criteria  
```javascript
db.potions.find(
    {
     "vendor": "Kettlecooked",
     "rating": 5
    }
)
```

Query with ranges  
```javascript
db.potions.find(
    {"price": {"$gt": 10, "$lt": 20 }}
)
```

Not equal to query  
```javascript
db.potions.find(
    {"vendor": {"$ne": "Brewers"}}
)
```

Match single element(s) in array matching both criterias  
_This will match anything with a size 8 < size < 16_
```javascript
db.potions.find(
    {"sizes": {"$elemMatch": {"$gt": 8, "$lt": 16}}}
)
```

Match any element in array matching the criterias but are not combined  
_This will match when a size > 8 and size < 16_
```javascript
db.potions.find(
    {"sizes": {"$gt": 8, "$lt": 16}}
)
```

Example of combing these queries  
```javascript
db.wands.find(
    {
        "maker": {"$ne": "Foxmond"},
        "level_required": {"$lte": 75},
        "price": {"$lt": 50},
        "lengths": {"$elemMatch": {"$gte": 3, "$lte": 4}}
    }
)
```

#### Quiery with filters

Only return certain fields  
_Note: You cannot combine true and false selectors, except for the id_  
```javascript
db.potions.find(
    {"grade": {"$gte": 80}},
    {"vendor": true, "price": true}
)
```

Do not include id  
_Note: the id is always shown unless specified not to. Also this is the only time you can have a true and false combination_  
```javascript
db.potions.find(
    {"grade": {"$gte": 80}},
    {"vendor": true, "price": true, "_id": false}
)
```

#### Using cursors

Use the `sort()` cursor  
_Note: -1 for descending and 1 for ascending_  
```javascript
db.potions.find().sort({"price": 1})
```

Use the `skip()` & `limit()`  
Here we get the first three documents  
```javascript
db.potions.find().limit(3)
```

Next we get the next three
```javascript
db.potions.find().skip(3).limit(3)
```



### Insert

Insert new document into collection `flag`  
```javascript
db.flags.insert({
    "name": "Web 1",
    "points": 10
})
```

When making a new document you can have an array and imbed another object like we did here with `damage`  
_Note: The embed document does not need and will not get its own object id since it is a child_  
```javascript
db.wands.insert({
    "name": "Dream Bender",
    "creator": "Foxmond",
    "level_required": 10,
    "price": 34.9,
    "powers": ["Fire", "Love"],
    "damage": {"magic": 4, "melee": 2}
})
```

### Remove

Remove matching a quiery  
```javascript
db.flags.remove({ "name": "Web 1" })
```

### Update

Update a document  
_Note: This will only update the first matching document_  
```javascript
db.flags.update(
    {"name": "Web 1"},
    {"$set": {"points": 5}}
)
```

_Note: If you just do not specifiy the `$set` operator then it will over write the document_  

Update multiple documents  
```javascript
db.flags.update(
    {"name": "Web 1"},
    {"$set": {"points": 5}},
    {"multi": true}
)
```

Update a count using `$inc`  
```javascript
db.results.update(
    {"teamname": "Netcats"},
    {"$inc": {"count": 1}}
)
```

Update a count in matching document and if it does not exist them make one  

```javascript
db.results.update(
    {"teamname": "Netcats"},
    {"$inc": {"count": 1}},
    {"upsert": true}
)
```

Unset a parameter in all documents  
```javascript
db.flags.update(
    {},
    {"$unset": {"value": ""}},
    {"multi": true}
)
```

Rename a field name  
```javascript
db.flags.update(
    {},
    {"$rename": {"value": "flag"}},
    {"multi": true}
)
```

Update value knowing position in array
```javascript
db.wands.update(
    {"ingredients": "secret"},
    {"$set": {"ingredients.0": 42}},
    {"multi": true}
)
```

Update value without knowing position in array using the `$`  
```javascript
db.wands.update(
    {"ingredients": "secret"},
    {"$set": {"ingredients.$": 42}},
    {"multi": true}
)
```

Update in a object  
```javascript
db.wands.insert(
    {"name": "Shrinking"},
    {"$set": {"rating.strength": 5}}
)
```

Remove last element in array  
_Note: To remove first element use -1 and 1 for last_  
```javascript
db.potions.update(
    {"name": "Shrinking"},
    {"$pop": {"categories": 1}}
)
```

Push into array  
_Note: If it exist it will still push_
```javascript
db.potions.update(
    {"name": "Shrinking"},
    {"$push": {"categories": "budget"}}
)
```

Push into array but only if it doesn't exist  
```javascript
db.potions.update(
    {"name": "Shrinking"},
    {"$addToSet": {"categories": "budget"}}
)
```

Remove a value from array  
_Note: This will remove every mataching element if it is not unique_  
```javascript
db.potions.update(
    {"name": "Shrinking"},
    {"$pull": {"categories": "tasty"}}
)
```

Multiply all documents by 10  
```javascript
db.wands.update(
    {},
    {"$mul": {"damage.melee": 10}},
    {"multi": true}
)
```

### Aggregation

Get service and the lastest status for that team
```javascript
db.results.aggregate(
    {"$match": {"type": "Service"}},
    {"$group":
        {
            "_id": {"type": "$group", "tnumber": "$teamnumber", "tname": "$teamname"},
            "status": {"$last": "$details"}
        }
    },
    {"$group":
        {
            "_id": "$_id.type",
            "teams": {"$addToSet": {"teamnumber": "$_id.teamnumber", "teamname": "$_id.tname", "status": "$status"}}
        }
    }
)
```

Same as above except there is no nested object
NOTE: Sort doesn't always sort for some reason in Go
```javascript
db.results.aggregate(
    {"$match": {"type": "Service"}},
    {"$group":
        {
            "_id": {"type": "$group", "tnumber": "$teamnumber", "tname": "$teamname"},
            "status": {"$last": "$details"}}
        },
    {"$group":
        {
            "_id": "$_id.type",
            "teams": {"$addToSet": {"number": "$_id.tnumber", "name": "$_id.tname", "status": "$status"}}
        }
    },
    {"$unwind": "$teams"},
    {"$project": {"_id": 0, "group": "$_id", "tnum": "$teams.number", "tname": "$teams.name", "status": "$teams.status"}},
    {"$sort": {"tnum": 1, "group": 1}}
)
```

Get service list form results
```javascript
db.results.aggregate(
    {"$match": {"type": "Service", "group": "web"}},
    {"$group": {"_id": "$group"}},
    {"$project": {"_id": 0, "group": "$_id"}}
)
```

Get By Serivce
```javascript
db.results.aggregate(
    {"$match": {"type": "Service", "group": "web"}},
    {"$group":
        {
            "_id": {"type": "$group", "tnumber": "$teamnumber", "tname": "$teamname"},
            "status": {"$last": "$details"}
        }
    },
    {"$project": {"_id": 0, "group": "$_id.type", "teamnumber": "$_id.tnumber", "teamname": "$_id.tname", "details": "$status"}},
    {"$sort": {"teamnumber": 1}}
)
```
