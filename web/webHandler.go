package web

import (
	"fmt"
	"net/http"
)

const loginPage = `
<h1>Login</h1>
<form method="post" action="/login">
	<label for="teamName">Team Name</label>
	<input type="text" id="teamName" name="teamName">
	<label for="password">Password</label>
	<input type="password" id="password" name="password">
	<button type="submit">Login</button>
</form>
`

func LoginPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, loginPage)
}

const teamPage = `
<h1>Team Page</h1>
<hr>
<small>Team: %s</small>
<form method="post" action="/logout">
	<button type="submit">Logout</button>
</form>
`

func TeamPage(w http.ResponseWriter, r *http.Request) {
	teamName := GetTeamName(r)
	fmt.Printf("%v\n", r)
	if teamName != "" {
		fmt.Fprintf(w, teamPage, teamName)
	} else {
		http.Redirect(w, r, "/loginPage", 302)
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	teamName := r.FormValue("teamName")
	password := r.FormValue("password")
	redirectTarget := "/loginPage"
	if teamName != "" && password != "" {
		// Check creds
		SetSession(teamName, w)
		redirectTarget = "/teamPage"
	}
	http.Redirect(w, r, redirectTarget, 302)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	ClearSession(w)
	http.Redirect(w, r, "/loginPage", 302)
}
