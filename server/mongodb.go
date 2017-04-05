package server

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var mongodbSession *mgo.Session

func init() {
	ServerCmd.PersistentFlags().String("mongodb_uri", "mongodb://127.0.0.1", "Address of MongoDB in use")
	viper.BindPFlag("database.mongodb_uri", ServerCmd.PersistentFlags().Lookup("mongodb_uri"))
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
		log.Println("Connected to mongodb:", mongodbSession.LiveServers())
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
		Logger.Println("No sessions available making new one")
		sessionCopy = DBSession().Copy()
	}
	collection = sessionCopy.DB(viper.GetString("database.dbname")).C(collectionName)
	return
}

func EnsureAdmin() {
	session, teamsCollection := GetSessionAndCollection("teams")
	defer session.Close()

	err := teamsCollection.Find(bson.M{"group": "admin"}).One(nil)
	if err == mgo.ErrNotFound {
		const adminAccName = "admin"
		admin := &Team{
			Name:   adminAccName,
			Group:  "admin",
			Number: -1,
			Ip:     "127.0.0.1",
		}
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
		admin.Hash = string(hashBytes)
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
