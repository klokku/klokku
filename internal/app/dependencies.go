package app

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klokku/klokku/internal/config"
	"github.com/klokku/klokku/internal/event_bus"
	"github.com/klokku/klokku/internal/utils"
	"github.com/klokku/klokku/pkg/budget_plan"
	"github.com/klokku/klokku/pkg/calendar"
	"github.com/klokku/klokku/pkg/calendar_provider"
	"github.com/klokku/klokku/pkg/clickup"
	"github.com/klokku/klokku/pkg/current_event"
	"github.com/klokku/klokku/pkg/stats"
	"github.com/klokku/klokku/pkg/user"
	"github.com/klokku/klokku/pkg/weekly_plan"
)

// Dependencies holds all services and handlers for the application.
type Dependencies struct {
	UserService user.Service
	UserHandler *user.Handler

	EventBus *event_bus.EventBus

	BudgetRepo        budget_plan.Repository
	BudgetPlanService budget_plan.Service
	BudgetPlanHandler *budget_plan.Handler

	WeeklyPlanRepo    weekly_plan.Repository
	WeeklyPlanService weekly_plan.Service
	WeeklyPlanHandler *weekly_plan.Handler

	KlokkuCalendarRepository calendar.Repository
	KlokkuCalendarService    *calendar.Service
	KlokkuCalendarHandler    *calendar.Handler

	CalendarProvider *calendar_provider.CalendarProvider

	CurrentEventRepo    current_event.Repository
	CurrentEventService current_event.Service
	CurrentEventHandler *current_event.EventHandler

	StatsService stats.StatsService
	StatsHandler *stats.StatsHandler

	ClickUpAuth    *clickup.ClickUpAuth
	ClickUpClient  clickup.Client
	ClickUpRepo    clickup.Repository
	ClickUpService *clickup.ServiceImpl
	ClickUpHandler *clickup.Handler

	Clock utils.Clock
}

// BuildDependencies initializes and wires all application services and handlers.
func BuildDependencies(db *pgxpool.Pool, cfg config.Application) *Dependencies {
	deps := &Dependencies{}

	deps.EventBus = event_bus.NewEventBus()

	deps.UserService = user.NewUserService(user.NewUserRepo(db))
	deps.UserHandler = user.NewHandler(deps.UserService)

	deps.BudgetRepo = budget_plan.NewBudgetPlanRepo(db)
	deps.BudgetPlanService = budget_plan.NewBudgetPlanService(deps.BudgetRepo, deps.EventBus)
	deps.BudgetPlanHandler = budget_plan.NewBudgetPlanHandler(deps.BudgetPlanService)
	deps.WeeklyPlanRepo = weekly_plan.NewRepo(db)
	deps.WeeklyPlanService = weekly_plan.NewService(deps.WeeklyPlanRepo, deps.BudgetPlanService, deps.EventBus)
	deps.WeeklyPlanHandler = weekly_plan.NewHandler(deps.WeeklyPlanService)

	deps.KlokkuCalendarRepository = calendar.NewRepository(db)
	deps.KlokkuCalendarService = calendar.NewService(deps.KlokkuCalendarRepository, deps.EventBus, deps.WeeklyPlanService.GetItemsForWeek)
	deps.KlokkuCalendarHandler = calendar.NewHandler(deps.KlokkuCalendarService)

	deps.CalendarProvider = calendar_provider.NewCalendarProvider(deps.UserService, deps.KlokkuCalendarService)

	deps.CurrentEventRepo = current_event.NewEventRepo(db)
	deps.CurrentEventService = current_event.NewEventService(deps.CurrentEventRepo, deps.CalendarProvider)
	deps.CurrentEventHandler = current_event.NewEventHandler(deps.CurrentEventService)

	deps.Clock = &utils.SystemClock{}
	deps.StatsService = stats.NewService(deps.CurrentEventService, deps.WeeklyPlanService, deps.BudgetPlanService, deps.CalendarProvider)
	deps.StatsHandler = stats.NewStatsHandler(deps.StatsService)

	deps.ClickUpAuth = clickup.NewClickUpAuth(db, deps.UserService, cfg)
	deps.ClickUpClient = clickup.NewClient(deps.ClickUpAuth)
	deps.ClickUpRepo = clickup.NewRepository(db)
	deps.ClickUpService = clickup.NewServiceImpl(deps.ClickUpRepo, deps.ClickUpClient)
	deps.ClickUpHandler = clickup.NewHandler(deps.ClickUpService, deps.ClickUpClient)

	return deps
}
