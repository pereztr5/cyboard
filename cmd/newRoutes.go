package cmd

import (
	"fmt"
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

func CreateWebRouter() *mux.Router {
	router := mux.NewRouter()
	// Public Routes
	//router.HandleFunc("/scores", Score)
	router.HandleFunc("/login", Login)
	// API Routes
	router.HandleFunc("/flags", GetFlags).Methods("GET")
	router.HandleFunc("/flags/verify", CheckFlag).Methods("POST")
	// Team Routes
	router.HandleFunc("/logout", Logout)
	// Static Files
	//router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	return router
}

func CreateTeamRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/team/teamPage", TeamPage)
	return router
}

/*
func Score(w http.ResponseWriter, r *http.Request) {
	scores, err := DataGetTeamScores()
	if err != nil {
		fmt.Fprintf(w, "Could not get team scores error: %s", err)
	}
	if err := json.NewEncoder(w).Encode(scores); err != nil {
		fmt.Fprintf(w, "Could not encode scores error: %s", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}
*/

func Login(w http.ResponseWriter, r *http.Request) {
	session := context.Get(r, "session").(*sessions.Session)
	switch {
	case r.Method == "GET":
		http.Redirect(w, r, "/login.html", 302)
	case r.Method == "POST":
		succ, err := CheckCreds(r)
		if err != nil {
			fmt.Println(err)
		}
		if succ {
			session.Save(r, w)
			http.Redirect(w, r, "/team/teamPage", 302)
		} else {
			http.Redirect(w, r, "/login.html", 302)
		}
	}
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session := context.Get(r, "session").(*sessions.Session)
	delete(session.Values, "id")
	http.Redirect(w, r, "/login.html", 302)
}

func TeamPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/teamPage.html", 302)
}
