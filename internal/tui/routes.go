package tui

type Route string

const (
	RouteSplash      Route = "splash"
	RouteMainMenu    Route = "main_menu"
	RouteMovieList   Route = "movie_list"
	RouteMovieDetail Route = "movie_detail"
	RouteStats       Route = "stats"
	RouteSettings    Route = "settings"
	RouteLogin       Route = "login"
	RouteHelp        Route = "help"
)

type Config struct {
	Theme       string
	ServerURL   string
	OfflineMode bool
}

type UserState struct {
	ID    string
	Email string
}

type AppState struct {
	Config Config
	User   UserState
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
