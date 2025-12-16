package app

import (
	"github.com/jackc/pgx/v5"
	"github.com/klokku/klokku/internal/config"
	"github.com/klokku/klokku/internal/event_bus"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/calendar_provider"
	"github.com/klokku/klokku/pkg/clickup"
	"github.com/klokku/klokku/pkg/event"
	"github.com/klokku/klokku/pkg/google"
	"github.com/klokku/klokku/pkg/stats"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
)

// Dependencies holds all services and handlers for the application.
type Dependencies struct {
	UserService user.Service
	UserHandler *user.Handler

	EventBus *event_bus.EventBus

	GoogleAuth    *google.GoogleAuth
	GoogleService google.Service
	GoogleHandler *google.Handler

	BudgetRepo        budget_plan.Repository
	BudgetService     budget_plan.Service
	BudgetPlanHandler *budget_plan.Handler

	WeeklyPlanRepo    weekly_plan.Repository
	WeeklyPlanService weekly_plan.Service
	WeeklyPlanHandler *weekly_plan.Handler

	KlokkuCalendarRepository calendar.Repository
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
func BuildDependencies(db *pgx.Conn, cfg config.Application) *Dependencies {
	deps := &Dependencies{}

	deps.EventBus = event_bus.NewEventBus()

	deps.UserService = user.NewUserService(user.NewUserRepo(db))
	deps.UserHandler = user.NewHandler(deps.UserService)

	deps.GoogleAuth = google.NewGoogleAuth(db, deps.UserService, cfg)
	deps.GoogleService = google.NewService(deps.GoogleAuth)
	deps.GoogleHandler = google.NewHandler(deps.GoogleService)

	deps.BudgetRepo = budget_plan.NewBudgetPlanRepo(db)
	deps.BudgetService = budget_plan.NewBudgetPlanService(deps.BudgetRepo, deps.EventBus)
	deps.BudgetPlanHandler = budget_plan.NewBudgetPlanHandler(deps.BudgetService)
	deps.WeeklyPlanRepo = weekly_plan.NewRepo(db)
	deps.WeeklyPlanService = weekly_plan.NewService(deps.WeeklyPlanRepo, deps.BudgetService, deps.EventBus)
	deps.WeeklyPlanHandler = weekly_plan.NewHandler(deps.WeeklyPlanService)

	deps.KlokkuCalendarRepository = calendar.NewRepository(db)
	deps.KlokkuCalendarService = calendar.NewService(deps.KlokkuCalendarRepository)
	deps.KlokkuCalendarHandler = calendar.NewHandler(deps.KlokkuCalendarService, deps.BudgetService.GetPlan)

	deps.CalendarProvider = calendar_provider.NewCalendarProvider(deps.UserService, deps.GoogleService, deps.KlokkuCalendarService)
	deps.CalendarMigrator = calendar_provider.NewEventsMigratorImpl(deps.CalendarProvider)
	deps.CalendarMigratorHandler = calendar_provider.NewMigratorHandler(deps.CalendarMigrator)

	deps.EventService = event.NewEventService(event.NewEventRepo(db), deps.CalendarProvider, deps.UserService)
	deps.EventHandler = event.NewEventHandler(deps.EventService)

	deps.Clock = &utils.SystemClock{}
	deps.EventStatsService = event.NewEventStatsServiceImpl(deps.CalendarProvider, deps.Clock)
	// TODO what is needed for the stats service?
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
