package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func GetFlags(w http.ResponseWriter, r *http.Request) {
	f, err := DataGetFlags()
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		fmt.Println("Error with DataGetFlags : %v", err)
		return
	}
	if err := json.NewEncoder(w).Encode(f); err != nil {
		http.Error(w, http.StatusText(500), 500)
		fmt.Println("Error encoding: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}

func CheckFlag(w http.ResponseWriter, r *http.Request) {
	challenge := r.FormValue("challenge")
	flag := r.FormValue("flag")
	var found bool
	var err error

	if len(flag) > 0 {
		found, err = DataCheckFlag(challenge, flag)
		// Logging found and not found flags
		if err != nil {
			// Once sessions work include the teamname here
			fmt.Printf("Team: team flag: %v\n", err)
		} else {
			fmt.Printf("Team: team flag: found\n")
		}
	}
	fmt.Fprint(w, found)
	w.WriteHeader(http.StatusOK)
}
