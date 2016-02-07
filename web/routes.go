package web

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
			"TeamPage",
			"GET",
			"/teamPage",
			TeamPage,
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

func CreateAPIRoutes(SE *ScoreEngineAPI) Routes {
	return Routes{
		Route{
			"GetFlags",
			"GET",
			"/flag/GetFlags",
			SE.GetFlags,
		},
		Route{
			"CheckFlag",
			"POST",
			"/flag/CheckFlag",
			SE.CheckFlag,
		},
	}
}
