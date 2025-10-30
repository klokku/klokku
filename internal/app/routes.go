package app

import (
	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/config"
)

// RegisterRoutes registers all API endpoints.
func RegisterRoutes(r *mux.Router, deps *Dependencies, cfg config.Application) {
	// Budget
	r.HandleFunc("/api/budget", deps.BudgetHandler.GetAll).Methods("GET")
	r.HandleFunc("/api/budget", deps.BudgetHandler.Register).Methods("POST")
	r.HandleFunc("/api/budget/{id}", deps.BudgetHandler.Update).Methods("PUT")
	r.HandleFunc("/api/budget/{id}/position", deps.BudgetHandler.SetPosition).Methods("PUT")
	r.HandleFunc("/api/budget/{id}", deps.BudgetHandler.Delete).Methods("DELETE")

	// Budget Override
	r.HandleFunc("/api/budget/override", deps.BudgetOverrideHandler.GetOverrides).Queries("startDate", "{startDate}").Methods("GET")
	r.HandleFunc("/api/budget/override", deps.BudgetOverrideHandler.Register).Methods("POST")
	r.HandleFunc("/api/budget/override/{overrideId}", deps.BudgetOverrideHandler.Update).Methods("PUT")
	r.HandleFunc("/api/budget/override/{overrideId}", deps.BudgetOverrideHandler.Delete).Methods("DELETE")

	// Events
	r.HandleFunc("/api/event", deps.EventHandler.StartEvent).Methods("POST")
	r.HandleFunc("/api/event/current/status", deps.EventHandler.FinishCurrentEvent).Methods("PATCH")
	r.HandleFunc("/api/event/current/start", deps.EventHandler.ModifyCurrentEventStartTime).Methods("PATCH")
	r.HandleFunc("/api/event/current", deps.EventHandler.DeleteCurrentEvent).Methods("DELETE")
	r.HandleFunc("/api/event/current", deps.EventHandler.GetCurrentEvent).Methods("GET")
	r.HandleFunc("/api/event", deps.EventHandler.GetLast5Events).Methods("GET").Queries("last", "5")

	// Stats
	r.HandleFunc("/api/stats", deps.StatsHandler.GetStats).Queries("fromDate", "{fromDate}", "toDate", "{toDate}").Methods("GET")

	// User management
	r.HandleFunc("/api/user", deps.UserHandler.CreateUser).Methods("POST")
	r.HandleFunc("/api/user/current", deps.UserHandler.CurrentUser).Methods("GET")
	r.HandleFunc("/api/user/current", deps.UserHandler.UpdateUser).Methods("PUT")
	r.HandleFunc("/api/user", deps.UserHandler.GetAvailableUsers).Methods("GET")
	r.HandleFunc("/api/user/{userId}", deps.UserHandler.DeleteUser).Methods("DELETE")
	r.HandleFunc("/api/user/current/photo", deps.UserHandler.UploadPhoto).Methods("PUT")
	r.HandleFunc("/api/user/current/photo", deps.UserHandler.GetPhoto).Methods("GET")
	r.HandleFunc("/api/user/{userId}/photo", deps.UserHandler.GetPhoto).Methods("GET")
	r.HandleFunc("/api/user/current/photo", deps.UserHandler.DeletePhoto).Methods("DELETE")

	// Klokku Calendar
	r.HandleFunc("/api/calendar/event", deps.KlokkuCalendarHandler.GetEvents).Queries("from", "{from}", "to", "{to}").Methods("GET")
	r.HandleFunc("/api/calendar/event", deps.KlokkuCalendarHandler.CreateEvent).Methods("POST")
	r.HandleFunc("/api/calendar/event/{eventUid}", deps.KlokkuCalendarHandler.UpdateEvent).Methods("PUT")
	r.HandleFunc("/api/calendar/event/{eventUid}", deps.KlokkuCalendarHandler.DeleteEvent).Methods("DELETE")
	r.HandleFunc("/api/calendar/import-from-google", deps.CalendarMigratorHandler.MigrateFromGoogleToKlokku).Queries("from", "{from}", "to", "{to}").Methods("POST")
	r.HandleFunc("/api/calendar/export-to-google", deps.CalendarMigratorHandler.MigrateFromKlokkuToGoogle).Queries("from", "{from}", "to", "{to}").Methods("POST")

	// Google integration
	r.HandleFunc("/api/integrations/google/auth/login", deps.GoogleAuth.OAuthLogin).Methods("GET")
	r.HandleFunc("/api/integrations/google/auth/logout", deps.GoogleAuth.OAuthLogout).Methods("DELETE")
	r.HandleFunc("/api/integrations/google/auth/callback", deps.GoogleAuth.OAuthCallback).Methods("GET")
	r.HandleFunc("/api/integrations/google/calendars", deps.GoogleHandler.ListCalendars).Methods("GET")

	// ClickUp integration
	r.HandleFunc("/api/integrations/clickup/auth/login", deps.ClickUpAuth.OAuthLogin).Methods("GET")
	r.HandleFunc("/api/integrations/clickup/auth/callback", deps.ClickUpAuth.OAuthCallback).Methods("GET")
	r.HandleFunc("/api/integrations/clickup/auth", deps.ClickUpAuth.IsAuthenticated).Methods("GET")
	r.HandleFunc("/api/integrations/clickup/auth", deps.ClickUpHandler.DisableIntegration).Methods("DELETE")
	r.HandleFunc("/api/integrations/clickup/workspace", deps.ClickUpHandler.ListWorkspaces).Methods("GET")
	r.HandleFunc("/api/integrations/clickup/space", deps.ClickUpHandler.ListSpaces).Queries("workspaceId", "{workspaceId}").Methods("GET")
	r.HandleFunc("/api/integrations/clickup/tag", deps.ClickUpHandler.ListTags).Queries("spaceId", "{spaceId}").Methods("GET")
	r.HandleFunc("/api/integrations/clickup/folder", deps.ClickUpHandler.ListFolders).Queries("spaceId", "{spaceId}").Methods("GET")
	r.HandleFunc("/api/integrations/clickup/configuration", deps.ClickUpHandler.GetConfiguration).Methods("GET")
	r.HandleFunc("/api/integrations/clickup/configuration", deps.ClickUpHandler.StoreConfiguration).Methods("PUT")
	r.HandleFunc("/api/integrations/clickup/tasks", deps.ClickUpHandler.GetTasks).Queries("budgetId", "{budgetId}").Methods("GET")
}
