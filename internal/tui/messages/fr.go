package messages

import (
	"fmt"

	"github.com/movietracker/movie-tracker/internal/domain"
)

// UI holds centralized French user-facing strings for the TUI.
var UI = struct {
	AppName string

	// Connection status (header)
	StatusOffline      string
	StatusOnlineSuffix string
	StatusDisconnected string

	// Sync footer
	SyncOffline   string
	SyncSyncing   string
	SyncError     string
	SyncPending   string
	SyncUpToDate  string
	SyncReady     string
	SyncPendingFmt   string
	SyncConflictsFmt string

	// Footer hints
	FooterDefault     string
	FooterSplash      string
	FooterMovieForm   string
	FooterMovieDetail string
	FooterStats       string
	FooterSettings    string
	FooterLogin       string
	FooterRegister    string
	FooterConflicts   string

	// Splash
	SplashWelcome string
	SplashTagline string
	SplashEnter   string

	// Movies list
	MoviesTitle           string
	SearchLabel           string
	FilterSortHint        string
	EmptyMovies           string
	EmptySearch           string
	AddMovieHint          string
	MovieFormHint         string
	MovieDetailHint       string
	SettingsHint          string
	LoginHint             string
	RegisterHint          string
	LoginLoading          string
	RegisterLoading       string
	LoginNoAccount        string

	// Forms
	AddMovieTitle    string
	TitleLabel       string
	YearLabel        string
	RatingLabel      string
	WatchedAtLabel   string
	ReviewLabel      string
	MovieDetailTitle string
	WatchedYes       string
	WatchedNo        string

	// Settings
	SettingsTitle       string
	ThemeLabel          string
	ServerURLLabel      string
	OfflineDisabled     string
	OfflineEnabled      string
	OfflineToggleHint   string
	ConnectedHintFmt    string
	BackupHint          string
	ExportHint          string
	BackupExportOK      string
	BackupImportOK      string
	BackupNeedAuth      string
	BackupUnavailable   string
	BackupExporting     string
	BackupImporting     string
	BackupLocalExportFmt string
	MoviesExportFmt     string
	ExportUnavailable   string
	ThemeChangedFmt     string

	// Stats
	StatsTitle          string
	StatsTotalMovies    string
	StatsTotalWatched   string
	StatsTotalRated     string
	StatsAverageRating  string
	StatsBestMovies     string
	StatsWorstMovies    string
	StatsByMonth        string
	StatsEmptyHint      string
	StatsRatingFmt      string

	// Auth screens
	LoginTitle    string
	RegisterTitle string
	EmailLabel    string
	PasswordLabel string
	ConfirmLabel  string

	// Menu
	MenuTitle string

	// Actions / feedback
	FilterFmt              string
	SortFmt                string
	FiltersReset           string
	MovieDeleted           string
	MovieAddedFmt          string
	DetailSaved            string
	WatchedToday           string
	Unwatched              string
	SettingsSaved          string
	LoggedOut              string
	ConnectedAsFmt         string
	AccountCreatedFmt      string
	SessionRestoredFmt     string
	ThemeEmpty             string
	ServerURLEmpty         string
	SaveFailedFmt          string
	AuthClientUnavailable  string
	PasswordMismatch       string
	MovieServiceUnavailable string
	NoMovieSelected        string
	LoadFailedFmt          string
	StatusIncompleteFmt    string
	StatsUnavailableFmt    string
	DeleteFailedFmt        string

	// Placeholders
	ThemePlaceholder     string
	ServerURLPlaceholder string
	EmailPlaceholder     string
	PasswordPlaceholder  string
	ConfirmPlaceholder   string
	TitlePlaceholder     string
	YearPlaceholder      string
	SearchPlaceholder    string
	RatingPlaceholder    string
	DatePlaceholder      string
	ReviewPlaceholder    string
	TMDBSearchPlaceholder string

	// TMDB
	TMDBSearchTitle      string
	TMDBSearchLabel      string
	TMDBResultsTitle     string
	TMDBSearchHint       string
	TMDBResultsHint      string
	TMDBSearching        string
	TMDBNoResults        string
	TMDBQueryTooShort    string
	TMDBUnavailable      string
	TMDBSelectedFmt      string
	ExternalIDFmt        string

	// Conflicts
	ConflictTitle            string
	ConflictScreenHint       string
	ConflictListHint         string
	ConflictEmpty            string
	ConflictResolved         string
	ConflictMovieFmt         string
	ConflictWatchFmt         string
	ConflictGenericFmt       string
	ConflictChoiceFmt        string
	ConflictLocalLabel       string
	ConflictRemoteLabel      string
	ConflictLocalPreviewFmt  string
	ConflictRemotePreviewFmt string
	ConflictNoPreview        string

	// Movie list status
	StatusUnwatched string

	// Help sections
	HelpTitle string
}{
	AppName: "MovieTracker",

	StatusOffline:      "hors ligne",
	StatusOnlineSuffix: " · en ligne",
	StatusDisconnected: "non connecté",

	SyncOffline:      "sync · hors ligne",
	SyncSyncing:      "sync · en cours...",
	SyncError:        "sync · erreur",
	SyncPending:      "sync · en attente",
	SyncUpToDate:     "sync · à jour",
	SyncReady:        "sync · prêt",
	SyncPendingFmt:   "sync · %d en attente",
	SyncConflictsFmt: "sync · %d conflit(s) — K pour résoudre",

	FooterDefault:     "↑/↓ naviguer • entrée sélectionner • / chercher • f filtre • t tri • a ajouter • S sync • K conflits • q quitter",
	FooterConflicts:   "↑/↓ conflit • tab local/distant • entrée résoudre • esc menu • q quitter",
	FooterSplash:      "entrée commencer • q quitter",
	FooterMovieForm:   "tab champ • ctrl+t recherche TMDB • entrée ajouter • esc retour • q quitter",
	FooterMovieDetail: "tab champ suivant • w vu aujourd'hui • u non vu • entrée enregistrer • esc liste",
	FooterStats:       "m menu • s paramètres • l connexion • S sync • ? aide • q quitter",
	FooterSettings:    "←/→ thème • tab champ • o hors ligne • e export serveur • i import serveur • E export local • j/J films JSON/CSV • entrée enregistrer • esc menu • q quitter",
	FooterLogin:       "tab champ suivant • entrée connexion • r inscription • esc menu • q quitter",
	FooterRegister:    "tab champ suivant • entrée inscription • esc connexion • q quitter",

	SplashWelcome: "Bienvenue dans MovieTracker",
	SplashTagline: "Une TUI pour suivre les films vus, les notes et les critiques.",
	SplashEnter:   "Appuie sur Entrée pour ouvrir le menu.",

	MoviesTitle:     "Films",
	SearchLabel:     "Recherche",
	FilterSortHint:  "Filtre : %s | Tri : %s | f filtre | t tri | c reset",
	EmptyMovies:     "Aucun film enregistré pour l'instant.",
	EmptySearch:     "Aucun film ne correspond à la recherche.",
	AddMovieHint:    "Appuie sur a pour ajouter un film.",
	MovieFormHint:   "Titre obligatoire, année optionnelle.",
	MovieDetailHint: "Modifie les champs puis valide avec Entrée.",
	SettingsHint:    "←/→ change le thème, e/i/E pour backup, j/J export films, Entrée pour enregistrer.",
	LoginHint:       "Connecte-toi au serveur MovieTracker.",
	RegisterHint:    "Crée un compte sur le serveur MovieTracker.",
	LoginLoading:    "Connexion en cours...",
	RegisterLoading: "Inscription en cours...",
	LoginNoAccount:  "Pas de compte ? Appuie sur r pour t'inscrire.",

	AddMovieTitle:    "Ajouter un film",
	TitleLabel:       "Titre",
	YearLabel:        "Année",
	RatingLabel:      "Note /10",
	WatchedAtLabel:   "Date de visionnage",
	ReviewLabel:      "Critique",
	MovieDetailTitle: "Détail film",
	WatchedYes:       "oui",
	WatchedNo:        "non",

	SettingsTitle:     "Paramètres",
	ThemeLabel:        "Thème",
	ServerURLLabel:    "URL serveur",
	OfflineDisabled:   "désactivé",
	OfflineEnabled:    "activé",
	OfflineToggleHint: "Mode hors ligne : %s (o pour basculer)",
	ConnectedHintFmt:     "Connecté : %s (d pour déconnecter)",
	BackupHint:           "Backup : e → serveur | i ← serveur | E → fichiers locaux",
	ExportHint:           "Export films : j → JSON | J → CSV",
	BackupExportOK:       "Configuration et état exportés vers le serveur.",
	BackupImportOK:       "Configuration et état importés depuis le serveur.",
	BackupNeedAuth:       "Connecte-toi au serveur pour utiliser le backup distant.",
	BackupUnavailable:    "Service de backup indisponible.",
	BackupExporting:      "Export vers le serveur...",
	BackupImporting:      "Import depuis le serveur...",
	BackupLocalExportFmt: "Fichiers JSON exportés dans %s",
	MoviesExportFmt:      "Films exportés dans %s",
	ExportUnavailable:    "Service d'export indisponible.",
	ThemeChangedFmt:      "Thème : %s",

	StatsTitle:         "Statistiques",
	StatsTotalMovies:   "Films suivis",
	StatsTotalWatched:  "Films vus",
	StatsTotalRated:    "Films notés",
	StatsAverageRating: "Note moyenne",
	StatsBestMovies:    "Meilleur(s) film(s)",
	StatsWorstMovies:   "Film(s) les moins aimés",
	StatsByMonth:       "Visionnages par mois",
	StatsEmptyHint:     "Ajoute des films pour voir tes statistiques.",
	StatsRatingFmt:     "%.1f / 10",

	LoginTitle:    "Connexion",
	RegisterTitle: "Inscription",
	EmailLabel:    "Email",
	PasswordLabel: "Mot de passe",
	ConfirmLabel:  "Confirmation",

	MenuTitle: "Menu principal",

	FilterFmt:               "Filtre : %s",
	SortFmt:                 "Tri : %s",
	FiltersReset:            "Recherche et filtres réinitialisés.",
	MovieDeleted:            "Film supprimé.",
	MovieAddedFmt:           "Film ajouté : %s",
	DetailSaved:             "Détail enregistré.",
	WatchedToday:            "Film marqué comme vu aujourd'hui.",
	Unwatched:               "Film marqué comme non vu.",
	SettingsSaved:           "Paramètres enregistrés.",
	LoggedOut:               "Déconnecté.",
	ConnectedAsFmt:          "Connecté en tant que %s",
	AccountCreatedFmt:       "Compte créé pour %s",
	SessionRestoredFmt:      "Session restaurée pour %s",
	ThemeEmpty:              "Le thème ne peut pas être vide.",
	ServerURLEmpty:          "L'URL du serveur ne peut pas être vide.",
	SaveFailedFmt:           "Sauvegarde impossible : %s",
	AuthClientUnavailable:   "Client d'authentification indisponible.",
	PasswordMismatch:        "Les mots de passe ne correspondent pas.",
	MovieServiceUnavailable: "Service films indisponible.",
	NoMovieSelected:         "Aucun film sélectionné.",
	LoadFailedFmt:           "Chargement impossible : %s",
	StatusIncompleteFmt:     "Statut incomplet : %s",
	StatsUnavailableFmt:     "Stats indisponibles : %s",
	DeleteFailedFmt:         "Suppression impossible : %s",

	ThemePlaceholder:     "midnight",
	ServerURLPlaceholder: "http://localhost:8080",
	EmailPlaceholder:     "vous@example.com",
	PasswordPlaceholder:  "mot de passe",
	ConfirmPlaceholder:   "confirmer",
	TitlePlaceholder:     "Titre du film",
	YearPlaceholder:      "2026",
	SearchPlaceholder:    "Rechercher un titre...",
	RatingPlaceholder:    "8.5",
	DatePlaceholder:      "YYYY-MM-DD",
	ReviewPlaceholder:     "Votre critique...",
	TMDBSearchPlaceholder: "Rechercher sur TMDB...",

	TMDBSearchTitle:     "Recherche TMDB",
	TMDBSearchLabel:     "Titre du film",
	TMDBResultsTitle:    "Résultats TMDB",
	TMDBSearchHint:      "Saisis un titre puis Entrée pour chercher. Esc pour revenir au formulaire.",
	TMDBResultsHint:     "↑/↓ sélectionner un résultat, Entrée pour confirmer.",
	TMDBSearching:       "Recherche TMDB en cours...",
	TMDBNoResults:       "Aucun résultat TMDB pour cette recherche.",
	TMDBQueryTooShort:   "Saisis au moins 2 caractères pour chercher.",
	TMDBUnavailable:     "Recherche TMDB indisponible (connexion ou clé API).",
	TMDBSelectedFmt:     "Film sélectionné : %s",
	ExternalIDFmt:         "Référence externe : %s",

	ConflictTitle:           "Conflits de synchronisation",
	ConflictScreenHint:      "Choisis la version à conserver pour chaque conflit.",
	ConflictListHint:        "Conflit en attente",
	ConflictEmpty:           "Aucun conflit de synchronisation.",
	ConflictResolved:        "Conflit résolu.",
	ConflictMovieFmt:        "Film : %s",
	ConflictWatchFmt:        "Suivi : %s",
	ConflictGenericFmt:      "%s · %s",
	ConflictChoiceFmt:       "Sélection : %s",
	ConflictLocalLabel:      "Version locale",
	ConflictRemoteLabel:     "Version distante",
	ConflictLocalPreviewFmt: "Local (%s) : %s",
	ConflictRemotePreviewFmt:"Distant (%s) : %s",
	ConflictNoPreview:       "aucun détail",

	StatusUnwatched: "non vu",

	HelpTitle: "Aide",
}

// FilterLabel returns the French label for a movie filter.
func FilterLabel(filter domain.MovieFilter) string {
	switch filter {
	case domain.MovieFilterWatched:
		return "vus"
	case domain.MovieFilterUnwatched:
		return "non vus"
	case domain.MovieFilterRated:
		return "notés"
	case domain.MovieFilterUnrated:
		return "sans note"
	default:
		return "tous"
	}
}

// SortLabel returns the French label for a movie sort order.
func SortLabel(sort domain.MovieSort) string {
	switch sort {
	case domain.MovieSortDate:
		return "date"
	case domain.MovieSortRating:
		return "note"
	default:
		return "titre"
	}
}

// OfflineModeLabel returns activé or désactivé for offline mode.
func OfflineModeLabel(enabled bool) string {
	if enabled {
		return UI.OfflineEnabled
	}
	return UI.OfflineDisabled
}

// SyncPendingLine formats the pending sync footer line.
func SyncPendingLine(count int) string {
	return fmt.Sprintf(UI.SyncPendingFmt, count)
}

// ThemeHeader formats the header right segment.
func ThemeHeader(theme, connection string) string {
	return fmt.Sprintf("theme %s | %s", theme, connection)
}
