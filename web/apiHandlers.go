package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type ScoreEngineAPI struct {
	myconnection *MongoConnection
}

type Flag struct {
	Name      string `json:"name"`
	Challenge string `json:"challenge"`
	Points    string `json:"poitns"`
	Value     string `json:"value"`
}

type Service struct {
	Team      string `json:"team"`
	Service   string `json:"serivce"`
	Timestamp string `json:"timestamp"`
	Ip        string `json:"ip"`
	Points    int    `json:"points"`
}

func NewScoreEngineAPI() *ScoreEngineAPI {
	SE := &ScoreEngineAPI{
		myconnection: NewDBConnection(),
	}
	return SE
}

func (Se *ScoreEngineAPI) GetFlags(w http.ResponseWriter, r *http.Request) {
	f, err := Se.myconnection.DataGetFlags()
	// Handle errors better
	if err != nil {
		fmt.Fprintf(w, "Could not get flags")
	}
	if err := json.NewEncoder(w).Encode(f); err != nil {
		fmt.Fprintf(w, "Could not get flags")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}

func (Se *ScoreEngineAPI) CheckFlag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challenge := vars["challenge"]
	flag := vars["value"]

	// Work around. Need to find out how to properly take care of this
	if len(flag) > 0 {
		// Need to handle these errors better
		// This needs to send the correct team who submitted the flag
		f, _ := Se.myconnection.DataCheckFlag(challenge, flag)
		fmt.Fprint(w, f)
		w.WriteHeader(http.StatusOK)
	} else {
		fmt.Fprint(w, "No Flag Entered")
	}
}
