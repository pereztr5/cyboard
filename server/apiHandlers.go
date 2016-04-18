package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/context"
)

func GetChallenges(w http.ResponseWriter, r *http.Request) {
	// TODO: For now this will only get one group of challenges
	chal, err := DataGetChallenges("Megatron")
	if err != nil {
		Logger.Printf("Error with DataGetChallenges: %v\n", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(chal); err != nil {
		Logger.Printf("Error encoding: %v\n", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func CheckFlag(w http.ResponseWriter, r *http.Request) {
	t := context.Get(r, "team").(Team)
	challenge := r.FormValue("challenge")
	flag := r.FormValue("flag")
	var found int
	var err error
	// Correct flag = 0
	// Wrong flag = 1
	// Got challenge already = 2
	if len(flag) > 0 {
		found, err = DataCheckFlag(t, Challenge{Name: challenge, Flag: flag})
		if err != nil {
			Logger.Printf("Error checking flag: %s for team: %s: %v\n", flag, t.Name, err)
		}
	}
	fmt.Fprint(w, found)
	w.WriteHeader(http.StatusOK)
}

func CheckAllFlags(w http.ResponseWriter, r *http.Request) {
	t := context.Get(r, "team").(Team)
	flag := r.FormValue("flag")
	var found int
	var err error
	// Correct flag = 0
	// Wrong flag = 1
	// Got challenge already = 2
	if len(flag) > 0 {
		found, err = DataCheckFlag(t, Challenge{Flag: flag})
		if err != nil {
			Logger.Printf("Error checking flag: %s for team: %s: %v\n", flag, t.Name, err)
		}
	}
	fmt.Fprint(w, found)
	w.WriteHeader(http.StatusOK)
}
