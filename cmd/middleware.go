package cmd

import (
	"fmt"
	"net/http"

	"github.com/gorilla/context"
	"gopkg.in/mgo.v2/bson"
)

// Code provided by https://github.com/elithrar
func CheckSessionID(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		http.Error(w, http.StatusText(400), 400)
		//http.Redirect(w, r, "/login.html", 302)
	}

	// Get the ID from the session (if it exists) and
	// look up the associated team
	id, ok := session.Values["id"]
	if !ok {
		// 403 - forbidden. No session ID! Invalid!
		http.Error(w, http.StatusText(403), 403)
		//http.Redirect(w, r, "/login.html", 302)
	} else {
		t, err := GetTeamById(id.(bson.ObjectId))
		if err != nil {
			// context.Set(r, "team", nil)
			// HTTP 500 here: GetTeamByID returning an error is probably a DB error?
			http.Error(w, http.StatusText(500), 500)
			//http.Redirect(w, r, "/login.html", 302)

			// Do you return a nil t and no error? Probably not. If you do, this applies.
			// Otherwise, just handle the error above.
			//if t == nil {
			//	// 403 - forbidden. The ID in the session isn't a valid team.
			//	http.Error(w, http.StatusText(403), 403)
			//	return
			//}

			// It's valid! Set it in the context to look it up later.
			context.Set(r, "team", t)
		}
	}

	next(w, r)
	// If you're using mux it calls this for you.
	// context.Clear(r)
}

func GetContext(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	session, err := Store.Get(r, "cyboard")
	if err != nil {
		fmt.Println("Get context", err)
	}
	context.Set(r, "session", session)
	fmt.Println(context.GetAll(r))
	if id, ok := session.Values["id"]; ok {
		t, err := GetTeamById(id.(bson.ObjectId))
		if err != nil {
			context.Set(r, "team", nil)
			fmt.Println(id)
		} else {
			context.Set(r, "team", t)
			fmt.Println(t.Teamname)
		}
	} else {
		context.Set(r, "team", nil)
	}
	next(w, r)
}

func RequireLogin(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	t := context.Get(r, "team")
	if t != nil {
		next(w, r)
	} else {
		http.Redirect(w, r, "/login.html", 302)
	}
}
