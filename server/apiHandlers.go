package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func GetPublicChallenges(w http.ResponseWriter, r *http.Request) {
	chal, err := DataGetChallenges(specialChallenges)
	if err != nil {
		Logger.Error("Error with DataGetChallenges: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(chal); err != nil {
		Logger.Error("Error encoding challenges: ", err)
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
			Logger.Errorf("Error checking flag: %s for team: %s: %v", flag, t.Name, err)
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
			Logger.Errorf("Error checking flag: %s for team: %s: %v", flag, t.Name, err)
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
		Logger.Error("Error with GetAllUsers: ", err)
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
		Logger.Error("Failed to update team: missing 'teamName' URL Paramater")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	var updateOp map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateOp); err != nil {
		Logger.Error("Error decoding update PUT body: ", err)
		http.Error(w, err.Error(), 500)
		return
	}

	if err := DataUpdateTeam(teamName, updateOp); err != nil {
		Logger.Error("Failed to update team: ", err)
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeleteTeam(w http.ResponseWriter, r *http.Request) {
	teamName, ok := mux.Vars(r)["teamName"]
	if !ok {
		Logger.Error("Failed to delete team: missing 'teamName' URL Paramater")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	err := DataDeleteTeam(teamName)
	if err != nil {
		Logger.Error("Failed to delete team: ", err)
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func CtfConfig(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team").(Team)
	if !AllowedToConfigureChallenges(t) {
		http.Error(w, http.StatusText(403), 403)
		return
	}

	chals, err := DataGetChallengesByGroup(t.AdminOf)
	if err != nil {
		Logger.Error("Failed to get flags by group: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(chals)
	if err != nil {
		Logger.Error("Error encoding CtfConfig json: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// todo(tbutts): Reduce copied code. Particularly in the Breakdown methods, and anything that returns JSON.
// todo(tbutts): Consider a middleware or some abstraction on the Json encoding (gorilla may already provide this)

func GetBreakdownOfSubmissionsPerFlag(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team").(Team)
	if !AllowedToConfigureChallenges(t) {
		http.Error(w, http.StatusText(403), 403)
		return
	}

	chalGroups := getChallengesOwnerOf(t.AdminOf, t.Group)

	flagsWithCapCounts, err := DataGetSubmissionsPerFlag(chalGroups)
	if err != nil {
		Logger.Error("Failed to get flags w/ occurences of capture: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(flagsWithCapCounts)
	if err != nil {
		Logger.Error("Error encoding FlagCaptures breakdown json: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func GetEachTeamsCapturedFlags(w http.ResponseWriter, r *http.Request) {
	t := r.Context().Value("team").(Team)
	if !AllowedToConfigureChallenges(t) {
		http.Error(w, http.StatusText(403), 403)
		return
	}

	chalGroups := getChallengesOwnerOf(t.AdminOf, t.Group)

	teamsWithCapturedFlags, err := DataGetEachTeamsCapturedFlags(chalGroups)
	if err != nil {
		Logger.Error("Failed to get each teams' flag captures: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(teamsWithCapturedFlags)
	if err != nil {
		Logger.Error("Error encoding each teams' flag captures breakdown json: ", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}
