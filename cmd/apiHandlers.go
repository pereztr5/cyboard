package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func GetFlags(w http.ResponseWriter, r *http.Request) {
	f, err := DataGetFlags()
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

func CheckFlag(w http.ResponseWriter, r *http.Request) {
	challenge := r.FormValue("challenge")
	flag := r.FormValue("flag")
	var found bool

	if len(flag) > 0 {
		// Need to handle these errors better
		// This needs to send the correct team who submitted the flag
		found, _ = DataCheckFlag(challenge, flag)
		fmt.Fprint(w, found)
		w.WriteHeader(http.StatusOK)
	} else {
		fmt.Fprint(w, found)
		w.WriteHeader(http.StatusOK)
	}
}
