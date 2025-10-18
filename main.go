package main

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/rest"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget"
	"github.com/klokku/klokku/pkg/budget_override"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/calendar_provider"
	"github.com/klokku/klokku/pkg/clickup"
	"github.com/klokku/klokku/pkg/event"
	"github.com/klokku/klokku/pkg/google"
	"github.com/klokku/klokku/pkg/stats"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // Import the SQLite driver
)

func init() {
	level := os.Getenv("LOG_LEVEL")
	if level != "" {
		logrusLevel, err := log.ParseLevel(level)
		if err != nil {
			log.Fatal(err)
		}
		log.SetLevel(logrusLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {

	db, err := sql.Open("sqlite", "./storage/klokku-data.db?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=1000000")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	m, err := migrate.New("file://migrations", "sqlite://./storage/klokku-data.db")
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(err)
	}

	router := mux.NewRouter()

	userService := user.NewUserService(user.NewUserRepo(db))
	userHandler := user.NewHandler(userService)

	googleAuth := google.NewGoogleAuth(db, userService)
	googleService := google.NewService(googleAuth)
	googleHandler := google.NewHandler(googleService)

	budgetRepo := budget.NewBudgetRepo(db)
	budgetService := budget.NewBudgetServiceImpl(budgetRepo)
	budgetOverrideRepo := budget_override.NewBudgetOverrideRepo(db)
	budgetOverrideService := budget_override.NewBudgetOverrideService(budgetOverrideRepo)

	klokkuCalendarRepository := calendar.NewRepository(db)
	klokkuCalendar := calendar.NewService(klokkuCalendarRepository)
	klokkuCalendarHandler := calendar.NewHandler(klokkuCalendar, budgetService.GetAll)

	calendarProvider := calendar_provider.NewCalendarProvider(userService, googleService, klokkuCalendar)
	calendarMigrator := calendar_provider.NewEventsMigratorImpl(calendarProvider)
	calendarMigratorHandler := calendar_provider.NewMigratorHandler(calendarMigrator)

	eventService := event.NewEventService(event.NewEventRepo(db), calendarProvider, userService)

	eventHandler := event.NewEventHandler(eventService)
	budgetHandler := budget.NewBudgetHandler(budgetService)
	budgetOverrideHandler := budget_override.NewBudgetOverrideHandler(budgetOverrideService)

	clock := &utils.SystemClock{}
	eventStatsService := event.NewEventStatsServiceImpl(calendarProvider, clock)
	statsService := stats.NewStatsServiceImpl(eventService, eventStatsService, budgetRepo, budgetOverrideRepo)
	csvStatsRenderer := stats.NewCsvStatsTransformer()
	statsHandler := stats.NewStatsHandler(statsService, csvStatsRenderer)

	clickUpAuth := clickup.NewClickUpAuth(db, userService)
	clickUpClient := clickup.NewClient(clickUpAuth)
	clickUpRepo := clickup.NewRepository(db)
	clickUpService := clickup.NewServiceImpl(clickUpRepo, clickUpClient)
	clickUpHandler := clickup.NewHandler(clickUpService, clickUpClient)

	router.Use(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userIdHeader := r.Header.Get("X-User-Id")
				ctx := r.Context()
				if userIdHeader != "" {
					ctx = user.WithId(r.Context(), userIdHeader)
				}
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})

	router.HandleFunc("/api/budget", budgetHandler.GetAll).Methods("GET")
	router.HandleFunc("/api/budget", budgetHandler.Register).Methods("POST")
	router.HandleFunc("/api/budget/{id}", budgetHandler.Update).Methods("PUT")
	router.HandleFunc("/api/budget/{id}/position", budgetHandler.SetPosition).Methods("PUT")
	router.HandleFunc("/api/budget/{id}", budgetHandler.Delete).Methods("DELETE")

	router.HandleFunc("/api/budget/override", budgetOverrideHandler.GetOverrides).Queries("startDate", "{startDate}").Methods("GET")
	router.HandleFunc("/api/budget/override", budgetOverrideHandler.Register).Methods("POST")
	router.HandleFunc("/api/budget/override/{overrideId}", budgetOverrideHandler.Update).Methods("PUT")
	router.HandleFunc("/api/budget/override/{overrideId}", budgetOverrideHandler.Delete).Methods("DELETE")

	router.HandleFunc("/api/event", eventHandler.StartEvent).Methods("POST")
	router.HandleFunc("/api/event/current/status", eventHandler.FinishCurrentEvent).Methods("PATCH")
	router.HandleFunc("/api/event/current/start", eventHandler.ModifyCurrentEventStartTime).Methods("PATCH")
	router.HandleFunc("/api/event/current", eventHandler.DeleteCurrentEvent).Methods("DELETE")
	router.HandleFunc("/api/event/current", eventHandler.GetCurrentEvent).Methods("GET")
	router.HandleFunc("/api/event", eventHandler.GetLast5Events).Methods("GET").Queries("last", "5")

	router.HandleFunc("/api/stats", statsHandler.GetStats).Queries("fromDate", "{fromDate}", "toDate", "{toDate}").Methods("GET")

	// Administration
	// User
	router.HandleFunc("/api/user", userHandler.CreateUser).Methods("POST")
	router.HandleFunc("/api/user/current", userHandler.CurrentUser).Methods("GET")
	router.HandleFunc("/api/user/current", userHandler.UpdateUser).Methods("PUT")
	router.HandleFunc("/api/user", userHandler.GetAvailableUsers).Methods("GET")
	router.HandleFunc("/api/user/{userId}", userHandler.DeleteUser).Methods("DELETE")
	router.HandleFunc("/api/user/current/photo", userHandler.UploadPhoto).Methods("PUT")
	router.HandleFunc("/api/user/current/photo", userHandler.GetPhoto).Methods("GET")
	router.HandleFunc("/api/user/{userId}/photo", userHandler.GetPhoto).Methods("GET")
	router.HandleFunc("/api/user/current/photo", userHandler.DeletePhoto).Methods("DELETE")

	// Calendar
	router.HandleFunc("/api/calendar/event", klokkuCalendarHandler.GetEvents).Queries("from", "{from}", "to", "{to}").Methods("GET")
	router.HandleFunc("/api/calendar/event", klokkuCalendarHandler.CreateEvent).Methods("POST")
	router.HandleFunc("/api/calendar/event/{eventUid}", klokkuCalendarHandler.UpdateEvent).Methods("PUT")
	router.HandleFunc("/api/calendar/event/{eventUid}", klokkuCalendarHandler.DeleteEvent).Methods("DELETE")
	router.HandleFunc("/api/calendar/import-from-google", calendarMigratorHandler.MigrateFromGoogleToKlokku).Queries("from", "{from}", "to", "{to}").Methods("POST")
	router.HandleFunc("/api/calendar/export-to-google", calendarMigratorHandler.MigrateFromKlokkuToGoogle).Queries("from", "{from}", "to", "{to}").Methods("POST")

	// Google Calendar Integration
	router.HandleFunc("/api/integrations/google/auth/login", googleAuth.OAuthLogin).Methods("GET")
	router.HandleFunc("/api/integrations/google/auth/logout", googleAuth.OAuthLogout).Methods("DELETE")
	router.HandleFunc("/api/integrations/google/auth/callback", googleAuth.OAuthCallback).Methods("GET")
	router.HandleFunc("/api/integrations/google/calendars", googleHandler.ListCalendars).Methods("GET")

	// ClickUp Integration
	router.HandleFunc("/api/integrations/clickup/auth/login", clickUpAuth.OAuthLogin).Methods("GET")
	router.HandleFunc("/api/integrations/clickup/auth/callback", clickUpAuth.OAuthCallback).Methods("GET")
	router.HandleFunc("/api/integrations/clickup/auth", clickUpAuth.IsAuthenticated).Methods("GET")
	router.HandleFunc("/api/integrations/clickup/auth", clickUpHandler.DisableIntegration).Methods("DELETE")
	router.HandleFunc("/api/integrations/clickup/workspace", clickUpHandler.ListWorkspaces).Methods("GET")
	router.HandleFunc("/api/integrations/clickup/space", clickUpHandler.ListSpaces).Queries("workspaceId", "{workspaceId}").Methods("GET")
	router.HandleFunc("/api/integrations/clickup/tag", clickUpHandler.ListTags).Queries("spaceId", "{spaceId}").Methods("GET")
	router.HandleFunc("/api/integrations/clickup/folder", clickUpHandler.ListFolders).Queries("spaceId", "{spaceId}").Methods("GET")
	router.HandleFunc("/api/integrations/clickup/configuration", clickUpHandler.GetConfiguration).Methods("GET")
	router.HandleFunc("/api/integrations/clickup/configuration", clickUpHandler.StoreConfiguration).Methods("PUT")
	router.HandleFunc("/api/integrations/clickup/tasks", clickUpHandler.GetTasks).Queries("budgetId", "{budgetId}").Methods("GET")

	if os.Getenv("KLOKKU_FRONTEND_DISABLED") != "true" {
		frontend := rest.NewFrontendHandler("frontend", "index.html")
		router.PathPrefix("/").Handler(frontend)
	}

	srv := &http.Server{
		Handler:      router,
		Addr:         ":8181",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Infof("Starting server on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())

}
