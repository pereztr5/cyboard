package models

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type Result struct {
	Id         bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Type       string        `json:"type" bson:"type"`
	Timestamp  time.Time     `json:"timestamp" bson:"timestamp"`
	Group      string        `json:"group" bson:"group"`
	Teamname   string        `json:"teamname" bson:"teamname"`
	Teamnumber int           `json:"teamnumber" bson:"teamnumber"`
	Details    string        `json:"details" bson:"details"`
	Points     int           `json:"points" bson:"points"`
}
