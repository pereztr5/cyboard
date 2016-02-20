package cmd

import "net/http"

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

func CreateWebRoutes() Routes {
	return Routes{
		Route{
			"LoginPage",
			"GET",
			"/loginPage",
			LoginPage,
		},
		Route{
			"Login",
			"POST",
			"/login",
			Login,
		},
		Route{
			"Logout",
			"POST",
			"/logout",
			Logout,
		},
	}
}

func CreateTeamRoutes() Routes {
	return Routes{
		Route{
			"GetFlags",
			"GET",
			"/flags",
			GetFlags,
		},
		Route{
			"CheckFlag",
			"POST",
			"/flag/verify",
			CheckFlag,
		},
		Route{
			"TeamPage",
			"GET",
			"/teamPage",
			TeamPage,
		},
	}
}
