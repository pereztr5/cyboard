package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
	"gopkg.in/mgo.v2"
)

var mongodbSession *mgo.Session

func init() {
	RootCmd.PersistentFlags().String("mongodb_uri", "mongodb://127.0.0.1", "Address of MongoDB in use")
	viper.BindPFlag("database.mongodb_uri", RootCmd.PersistentFlags().Lookup("mongodb_uri"))
	CreateUniqueIndexes()
}

func DBSession() *mgo.Session {
	if mongodbSession == nil {
		fmt.Println("Making new MongoDB Session")
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

func Flags() *mgo.Collection {
	return DB().C("flags")
}

func CreateUniqueIndexes() {
	inx := mgo.Index{
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}

	teamInx := inx
	teamInx.Key = []string{"teamname"}

	flagInx := inx
	flagInx.Key = []string{"flagname"}

	if err := Teams().EnsureIndex(teamInx); err != nil {
		fmt.Println(err)
	}
	if err := Flags().EnsureIndex(flagInx); err != nil {
		fmt.Println(err)
	}
}

func GetSessionAndCollection(collectionName string) (sessionCopy *mgo.Session, collection *mgo.Collection) {
	if mongodbSession != nil {
		sessionCopy = mongodbSession.Copy()
	} else {
		fmt.Println("No sessions available making new one")
		sessionCopy = DBSession().Copy()
	}
	collection = sessionCopy.DB("scorengine").C(collectionName)
	return
}
