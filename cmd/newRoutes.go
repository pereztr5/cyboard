package cmd

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func CreateWebRouter() *mux.Router {
	router := mux.NewRouter()
	// Public Routes
	// Static Files
	//router.Handle("/", http.FileServer(http.Dir("./static/")))
	//router.HandleFunc("/scores", Score)
	router.HandleFunc("/login", ShowLogin).Methods("GET")
	router.HandleFunc("/login", SubmitLogin).Methods("POST")
	// API Routes
	router.HandleFunc("/flags", GetFlags).Methods("GET")
	router.HandleFunc("/flags/verify", CheckFlag).Methods("POST")
	// Team Routes
	router.HandleFunc("/logout", Logout)

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

func ShowLogin(w http.ResponseWriter, r *http.Request) {
	// TODO(pereztr5): Render a template instead of re-directing to a static page
	http.Redirect(w, r, "/login.html", 302)
}

func SubmitLogin(w http.ResponseWriter, r *http.Request) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		log.Printf("Getting from Store failed: %v", err)
		http.Error(w, http.StatusText(400), 400)
		return
	}

	succ, err := CheckCreds(w, r)
	if err != nil {
		// Print a more verbose error message for debugging purposes
		log.Printf("CheckCreds failed: %v", err)
		http.Error(w, http.StatusText(403), 403)
		return
	}

	if succ {
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}

		http.Redirect(w, r, "/team/teamPage", 302)
		return
	}

	http.Redirect(w, r, "/login.html", 302)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		http.Error(w, http.StatusText(400), 400)
		return
	}

	delete(session.Values, "id")
	// Make sure we save the session after deleting the ID.
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	http.Redirect(w, r, "/login.html", 302)
}

func TeamPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/teamPage.html", 302)
}
