package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/context"
)

func GetFlags(w http.ResponseWriter, r *http.Request) {
	f, err := DataGetFlags()
	if err != nil {
		Logger.Printf("Error with DataGetFlags: %v\n", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	if err := json.NewEncoder(w).Encode(f); err != nil {
		Logger.Printf("Error encoding: %v\n", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}

func CheckFlag(w http.ResponseWriter, r *http.Request) {
	t := context.Get(r, "team").(Team)
	challenge := r.FormValue("challenge")
	flag := r.FormValue("flag")
	var found int
	// Correct flag = 0
	// Wrong flag = 1
	// Has flag = 2
	var err error
	if len(flag) > 0 {
		for _, f := range t.Flags {
			if challenge == f.Challenge {
				found = 2
				break
			}
		}
		if found != 2 {
			found, err = DataCheckFlag(t.Teamname, challenge, flag)
			if err != nil {
				Logger.Printf("Error checking flag: %s for team: %s: %v\n", flag, t.Teamname, err)
			}
		}
	}
	fmt.Fprint(w, found)
	w.WriteHeader(http.StatusOK)
}
