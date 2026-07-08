package tui

type Route string

const (
	RouteSplash      Route = "splash"
	RouteMainMenu    Route = "main_menu"
	RouteMovieList   Route = "movie_list"
	RouteMovieForm   Route = "movie_form"
	RouteMovieDetail Route = "movie_detail"
	RouteStats       Route = "stats"
	RouteSettings    Route = "settings"
	RouteLogin       Route = "login"
	RouteRegister    Route = "register"
	RouteHelp        Route = "help"
)

const localUserID = "local-user"

type Config struct {
	Theme       string
	ServerURL   string
	OfflineMode bool
}

type SessionState struct {
	AccessToken   string
	RefreshToken  string
	ServerUserID  string
	Email         string
	Authenticated bool
}

type AppState struct {
	Config  Config
	Session SessionState
}

func defaultState() AppState {
	return AppState{
		Config: Config{
			Theme:       "midnight",
			ServerURL:   "http://localhost:8080",
			OfflineMode: true,
		},
	}
}

func effectiveUserID(state AppState) string {
	if state.Session.Authenticated && state.Session.ServerUserID != "" {
		return state.Session.ServerUserID
	}
	return localUserID
}