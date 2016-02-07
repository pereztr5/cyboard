package web

import (
	"fmt"
	"net/http"

	"github.com/gorilla/securecookie"
)

var CookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32),
)

func SetSession(teamName string, w http.ResponseWriter) {
	value := map[string]string{
		"teamName": teamName,
	}

	if encoded, err := CookieHandler.Encode("session", value); err == nil {
		cookie := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}
}

func GetTeamName(r *http.Request) (teamName string) {
	if cookie, err := r.Cookie("session"); err == nil {
		cookieValue := make(map[string]string)
		if err = CookieHandler.Decode("session", cookie.Value, &cookieValue); err == nil {
			teamName = cookieValue["teamName"]
		}
	} else {
		fmt.Printf("%v\n", err)
	}
	return teamName
}

func ClearSession(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/loginPage",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}
