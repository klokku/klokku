package app

import (
	"database/sql"

	"github.com/klokku/klokku/internal/auth"
	"github.com/klokku/klokku/internal/config"
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
)

// Dependencies holds all services and handlers for the application.
type Dependencies struct {
	AuthTokenValidator auth.TokenValidator

	UserService user.Service
	UserHandler *user.Handler

	GoogleAuth    *google.GoogleAuth
	GoogleService google.Service
	GoogleHandler *google.Handler

	BudgetRepo            budget.BudgetRepo
	BudgetService         *budget.BudgetServiceImpl
	BudgetHandler         *budget.BudgetHandler
	BudgetOverrideRepo    budget_override.BudgetOverrideRepo
	BudgetOverrideService *budget_override.BudgetOverrideServiceImpl
	BudgetOverrideHandler *budget_override.BudgetOverrideHandler

	KlokkuCalendarRepository *calendar.RepositoryImpl
	KlokkuCalendarService    *calendar.Service
	KlokkuCalendarHandler    *calendar.Handler

	CalendarProvider        *calendar_provider.CalendarProvider
	CalendarMigrator        *calendar_provider.EventsMigratorImpl
	CalendarMigratorHandler *calendar_provider.MigratorHandler

	EventService event.EventService
	EventHandler *event.EventHandler

	EventStatsService event.EventStatsService
	StatsService      *stats.StatsServiceImpl
	CsvStatsRenderer  *stats.CsvStatsRendererImpl
	StatsHandler      *stats.StatsHandler

	ClickUpAuth    *clickup.ClickUpAuth
	ClickUpClient  clickup.Client
	ClickUpRepo    clickup.Repository
	ClickUpService *clickup.ServiceImpl
	ClickUpHandler *clickup.Handler

	Clock utils.Clock
}

// BuildDependencies initializes and wires all application services and handlers.
func BuildDependencies(db *sql.DB, cfg config.Application) *Dependencies {
	deps := &Dependencies{}

	deps.AuthTokenValidator = auth.TokenValidator{}

	deps.UserService = user.NewUserService(user.NewUserRepo(db))
	deps.UserHandler = user.NewHandler(deps.UserService)

	deps.GoogleAuth = google.NewGoogleAuth(db, deps.UserService, cfg)
	deps.GoogleService = google.NewService(deps.GoogleAuth)
	deps.GoogleHandler = google.NewHandler(deps.GoogleService)

	deps.BudgetRepo = budget.NewBudgetRepo(db)
	deps.BudgetService = budget.NewBudgetServiceImpl(deps.BudgetRepo)
	deps.BudgetHandler = budget.NewBudgetHandler(deps.BudgetService)
	deps.BudgetOverrideRepo = budget_override.NewBudgetOverrideRepo(db)
	deps.BudgetOverrideService = budget_override.NewBudgetOverrideService(deps.BudgetOverrideRepo)
	deps.BudgetOverrideHandler = budget_override.NewBudgetOverrideHandler(deps.BudgetOverrideService)

	deps.KlokkuCalendarRepository = calendar.NewRepository(db)
	deps.KlokkuCalendarService = calendar.NewService(deps.KlokkuCalendarRepository)
	deps.KlokkuCalendarHandler = calendar.NewHandler(deps.KlokkuCalendarService, deps.BudgetService.GetAll)

	deps.CalendarProvider = calendar_provider.NewCalendarProvider(deps.UserService, deps.GoogleService, deps.KlokkuCalendarService)
	deps.CalendarMigrator = calendar_provider.NewEventsMigratorImpl(deps.CalendarProvider)
	deps.CalendarMigratorHandler = calendar_provider.NewMigratorHandler(deps.CalendarMigrator)

	deps.EventService = event.NewEventService(event.NewEventRepo(db), deps.CalendarProvider, deps.UserService)
	deps.EventHandler = event.NewEventHandler(deps.EventService)

	deps.Clock = &utils.SystemClock{}
	deps.EventStatsService = event.NewEventStatsServiceImpl(deps.CalendarProvider, deps.Clock)
	deps.StatsService = stats.NewStatsServiceImpl(deps.EventService, deps.EventStatsService, deps.BudgetRepo, deps.BudgetOverrideRepo)
	deps.CsvStatsRenderer = stats.NewCsvStatsTransformer()
	deps.StatsHandler = stats.NewStatsHandler(deps.StatsService, deps.CsvStatsRenderer)

	deps.ClickUpAuth = clickup.NewClickUpAuth(db, deps.UserService, cfg)
	deps.ClickUpClient = clickup.NewClient(deps.ClickUpAuth)
	deps.ClickUpRepo = clickup.NewRepository(db)
	deps.ClickUpService = clickup.NewServiceImpl(deps.ClickUpRepo, deps.ClickUpClient)
	deps.ClickUpHandler = clickup.NewHandler(deps.ClickUpService, deps.ClickUpClient)

	return deps
}
