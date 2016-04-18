package server

import (
	"log"
	"os"

	"github.com/spf13/viper"
	"gopkg.in/mgo.v2"
)

var mongodbSession *mgo.Session

func init() {
	ServerCmd.PersistentFlags().String("mongodb_uri", "mongodb://127.0.0.1", "Address of MongoDB in use")
	viper.BindPFlag("database.mongodb_uri", ServerCmd.PersistentFlags().Lookup("mongodb_uri"))
	CreateUniqueIndexes()
}

func DBSession() *mgo.Session {
	if mongodbSession == nil {
		log.Println("Making new MongoDB Session")
		uri := os.Getenv("MONGODB_URI")
		if uri == "" {
			uri = viper.GetString("database.mongodb_uri")
			if uri == "" {
				log.Fatalln("No connection uri for MongoDB provided")
			}
		}
		var err error
		mongodbSession, err = mgo.Dial(uri)
		if mongodbSession == nil || err != nil {
			log.Fatalf("Can't connect to MongoDB, go error %v\n", err)
		}
		mongodbSession.SetSafe(&mgo.Safe{})
	}
	return mongodbSession
}

func DB() *mgo.Database {
	return DBSession().DB(viper.GetString("database.dbname"))
}

func Teams() *mgo.Collection {
	return DB().C("teams")
}

func Challenges() *mgo.Collection {
	return DB().C("challenges")
}

func CreateUniqueIndexes() {
	inx := mgo.Index{
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	teamInx := inx
	teamInx.Key = []string{"number", "name"}

	chalInx := inx
	chalInx.Key = []string{"name"}

	if err := Teams().EnsureIndex(teamInx); err != nil {
		Logger.Println(err)
	}
	if err := Challenges().EnsureIndex(chalInx); err != nil {
		Logger.Println(err)
	}
}

func GetSessionAndCollection(collectionName string) (sessionCopy *mgo.Session, collection *mgo.Collection) {
	if mongodbSession != nil {
		sessionCopy = mongodbSession.Copy()
	} else {
		log.Println("No sessions available making new one")
		sessionCopy = DBSession().Copy()
	}
	collection = sessionCopy.DB("scorengine").C(collectionName)
	return
}
