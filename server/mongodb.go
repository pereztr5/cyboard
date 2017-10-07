package server

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	mongodbSession    *mgo.Session
	dbSettings        *DBSettings
	specialChallenges []string
)

func DBSession() *mgo.Session {
	if mongodbSession == nil {
		Logger.Println("Making new MongoDB Session")
		uri := os.Getenv("MONGODB_URI")
		if uri == "" {
			uri = dbSettings.URI
			if uri == "" {
				Logger.Fatalln("No connection uri for MongoDB provided")
			}
		}
		var err error
		mongodbSession, err = mgo.Dial(uri)
		if mongodbSession == nil || err != nil {
			Logger.Fatalln("Can't connect to MongoDB: ", err)
		}
		mongodbSession.SetSafe(&mgo.Safe{})
		Logger.Println("Connected to mongodb:", mongodbSession.LiveServers())
	}
	return mongodbSession
}

func DB() *mgo.Database {
	return DBSession().DB(dbSettings.DBName)
}

func Teams() *mgo.Collection {
	return DB().C("teams")
}

func Challenges() *mgo.Collection {
	return DB().C("challenges")
}

func CreateIndexes() {
	inx := mgo.Index{
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	// Unique + sparse indexes enforce that, across each doc,
	// there are no duplicates of the Key fields

	teamUniqInx := inx
	teamUniqInx.Key = []string{"number", "name"}

	chalUniqInx := inx
	chalUniqInx.Key = []string{"name"}

	// Comments below indicate at least one method the index is used in (there may be more)

	collectionToIndexesMap := map[string][]mgo.Index{
		"teams": {
			teamUniqInx,
			{Key: []string{"group", "number"}}, // DataGetAllUsers
		},
		"challenges": {
			chalUniqInx,
			{Key: []string{"group"}},        // DataGetChallenges
			{Key: []string{"flag", "name"}}, // DataCheckFlag
		},
		"results": {
			{Key: []string{"type", "group"}},       // DataGetResultByService
			{Key: []string{"teamname", "details"}}, // HasFlag
			{Key: []string{"teamname", "type"}},    // DataGetTeamChallenges
			{Key: []string{"group"}},               // DataGetSubmissionsPerFlag & DataGetEachTeamsCapturedFlags
		},
	}

	for coll, indexes := range collectionToIndexesMap {
		for _, inx := range indexes {
			if err := DB().C(coll).EnsureIndex(inx); err != nil {
				Logger.Fatalf("In collection '%v': failed ensuring index %v: %v", coll, inx.Key, err)
			}
		}
	}
}

func GetSessionAndCollection(collectionName string) (sessionCopy *mgo.Session, collection *mgo.Collection) {
	if mongodbSession != nil {
		sessionCopy = mongodbSession.Copy()
	} else {
		Logger.Println("No sessions available making new one")
		sessionCopy = DBSession().Copy()
	}
	collection = sessionCopy.DB(dbSettings.DBName).C(collectionName)
	return
}

// SetupMongo copies settings into app-wide vars used for connecting & interacting with MongoDB
func SetupMongo(settings *DBSettings, special []string) {
	// If the DBName is empty, the mongo driver will fallback to `test`, but we'd rather fail fast.
	if settings.DBName == "" {
		Logger.Fatal(`DB name is empty: database.dbname=""`)
	}

	s := *settings
	dbSettings = &s

	specialChallenges = special
}

func EnsureAdmin() {
	session, teamsCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err := teamsCollection.Find(bson.M{"group": "admin"}).One(nil)
	if err == mgo.ErrNotFound {
		const adminAccName = "admin"
		// Read initial password from command line
		fmt.Printf("*** No previously configured admin user found ***\n"+
			"Setting up '%s' account.\n"+
			"Provide a password for the account "+
			"(you can change it later on the website):\n",
			adminAccName)
		fmt.Print(">> ")
		pass, err := ReadStdinLine()
		if err != nil {
			Logger.Fatal("Failed to read pass:", err)
		}
		hashBytes, err := bcrypt.GenerateFromPassword(pass, bcrypt.DefaultCost)
		if err != nil {
			Logger.Fatal("Failed to hash password:", err)
		}
		admin := &Team{
			Name:   adminAccName,
			Group:  "admin",
			Number: -1,
			Ip:     "127.0.0.1",
			Hash:   string(hashBytes),
		}
		err = teamsCollection.Insert(admin)
		if err != nil {
			Logger.Fatal("Failed to add admin to MongoDB:", err)
		}
		fmt.Printf("'%s' account configured.\n", adminAccName)
		fmt.Print("Log in on the website to finish other configurations\n")
	} else if err != nil {
		Logger.Fatal("Failed to query mongo for admin user:", err)
	}
}
