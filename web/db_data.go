package web

import (
	"errors"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const CONNECTIONSTRING = "mongodb://127.0.0.1"

type flagDocument struct {
	//Id        bson.ObjectId `bson:"_id"`
	Name      string `json:"name"`
	Challenge string `json:"challenge"`
	Points    int    `json:"points"`
	//Value     string        `json:"value"`
}

type flagSmallDocument struct {
	Name   string `json:"name"`
	Points int    `json:"points"`
}

type teamScoreDocument struct {
	Team  string `json:"team"`
	Score int    `json:"score"`
}

type ServiceDocument struct {
	//Id        bson.ObjectId `bson:"_id"`
	Team      string `json:"team"`
	Service   string `json:"serivce"`
	Timestamp string `json:"timestamp"`
	Ip        string `json:"ip"`
	Points    int    `json:"points"`
}

type MongoConnection struct {
	originalSession *mgo.Session
}

func NewDBConnection() (conn *MongoConnection) {
	conn = new(MongoConnection)
	conn.createLocalConnection()
	return
}

func (c *MongoConnection) createLocalConnection() (err error) {
	fmt.Println("Connecting to local mongo server....")
	c.originalSession, err = mgo.Dial(CONNECTIONSTRING)
	if err == nil {
		fmt.Println("Connection established to mongo server")
		flagcollection := c.originalSession.DB("scorengine").C("flags")
		if flagcollection == nil {
			err = errors.New("Collection could not be created, maybe need to create it manually")
		}
		//This will create a unique index to ensure that there won't be duplicate shorturls in the database.
		index := mgo.Index{
			Key:      []string{"$text:key"},
			Unique:   true,
			DropDups: true,
		}
		flagcollection.EnsureIndex(index)
	} else {
		fmt.Printf("Error occured while creating mongodb connection: %s", err.Error())
	}
	return
}

func (c *MongoConnection) CloseConnection() {
	if c.originalSession != nil {
		c.originalSession.Close()
	}
}

func (c *MongoConnection) getSessionAndCollection(collection string) (session *mgo.Session, flagCollection *mgo.Collection, err error) {
	if c.originalSession != nil {
		session = c.originalSession.Copy()
		flagCollection = session.DB("scorengine").C(collection)
	} else {
		err = errors.New("No original session found")
	}
	return
}

// Query DB statements
/*
 *	DataFindFlag is not admins only
 *
 */
func (c *MongoConnection) DataGetFlags() ([]flagDocument, error) {
	//result := flagDocument{}
	result := []flagDocument{}
	session, flagCollection, err := c.getSessionAndCollection("flags")
	if err != nil {
		return result, err
	}
	defer session.Close()
	err = flagCollection.Find(nil).Select(bson.M{"_id": 0, "value": 0}).All(&result)
	if err != nil {
		fmt.Printf("Could not get flags\n")
	}
	return result, nil
}

func (c *MongoConnection) DataCheckFlag(chal, val string) (bool, error) {
	var found bool
	result := flagSmallDocument{}
	session, flagCollection, err := c.getSessionAndCollection("flags")
	if err != nil {
		return found, err
	}
	defer session.Close()
	err = flagCollection.Find(bson.M{"challenge": chal, "value": val}).Select(bson.M{"name": 1, "points": 1}).One(&result)
	if err != nil {
		fmt.Printf("Invalid flag\n")
	} else {
		// Need to add points to the team who got the flag
		found = true
	}
	return found, nil
}
