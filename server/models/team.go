package models

import "gopkg.in/mgo.v2/bson"

type Team struct {
	Id      bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Group   string        `json:"group"`
	Number  int           `json:"number"`
	Name    string        `json:"name"`
	Ip      string        `json:"ip"`
	Hash    string        `json:"-"`
	AdminOf string        `json:"adminof"`
}
