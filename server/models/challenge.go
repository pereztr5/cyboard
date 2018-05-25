package models

import "gopkg.in/mgo.v2/bson"

type Challenge struct {
	Id          bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Group       string        `json:"group"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Flag        string        `json:"flag" bson:"flag"`
	Points      int           `json:"points"`
}
