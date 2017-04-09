package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func GetChallenges(w http.ResponseWriter, r *http.Request) {
	// TODO: For now this will only get one group of challenges
	chal, err := DataGetChallenges("Rusted Bunions")
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
	t := r.Context().Value("team").(Team)
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
	t := r.Context().Value("team").(Team)
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

func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	users := DataGetAllUsers()
	err := json.NewEncoder(w).Encode(users)
	if err != nil {
		Logger.Println("Error with GetAllUsers:", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func AddTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := ParseTeamCsv(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	// The unique indexes on the teams collection
	// will prevent bad duplicates.
	err = DataAddTeams(teams)
	if err != nil {
		// MongoDB errors should be safe to give back to admins.
		http.Error(w, err.Error(), http.StatusExpectationFailed)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func UpdateTeam(w http.ResponseWriter, r *http.Request) {
	teamName, ok := mux.Vars(r)["teamName"]
	if !ok {
		Logger.Println("Failed to update team: missing 'teamName' URL Paramater")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	var updateOp map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateOp); err != nil {
		Logger.Println("Error decoding update PUT body:", err)
		http.Error(w, err.Error(), 500)
		return
	}

	if err := DataUpdateTeam(teamName, updateOp); err != nil {
		Logger.Println("Failed to update team:", err)
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeleteTeam(w http.ResponseWriter, r *http.Request) {
	teamName, ok := mux.Vars(r)["teamName"]
	if !ok {
		Logger.Println("Failed to delete team: missing 'teamName' URL Paramater")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	err := DataDeleteTeam(teamName)
	if err != nil {
		Logger.Println("Failed to delete team:", err)
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}
