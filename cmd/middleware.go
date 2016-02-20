package cmd

import (
	"fmt"
	"net/http"

	"github.com/gorilla/context"
	"gopkg.in/mgo.v2/bson"
)

func GetContext(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing request", http.StatusInternalServerError)
	}
	session, _ := Store.Get(r, "cyboard")
	fmt.Println("Get", session)
	context.Set(r, "session", session)
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
	context.Clear(r)
}

func RequireLogin(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	t := context.Get(r, "team")
	fmt.Println("RequireLogin", t)
	if t != nil {
		next(w, r)
	} else {
		http.Redirect(w, r, "/login.html", 302)
	}
}
