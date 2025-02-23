package stats

import (
	"github.com/klokku/klokku/pkg/budget"
	"github.com/klokku/klokku/pkg/budget_override"
	"testing"
	"time"
)

var startDate = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
var endDate = time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)
var budget1 = budget.Budget{
	ID:         1,
	Name:       "Budget 1",
	WeeklyTime: time.Duration(300) * time.Minute,
}
var budget2 = budget.Budget{
	ID:         2,
	Name:       "Budget 2",
	WeeklyTime: time.Duration(240) * time.Minute,
}
var budget1Override = budget_override.BudgetOverride{
	ID:         234,
	BudgetID:   1,
	StartDate:  startDate,
	WeeklyTime: time.Duration(150) * time.Minute,
}
var budget2Override = budget_override.BudgetOverride{
	ID:         123,
	BudgetID:   2,
	StartDate:  startDate,
	WeeklyTime: time.Duration(360) * time.Minute,
}

func TestCsvStatsRendererImpl_RenderStats(t1 *testing.T) {
	type args struct {
		stats StatsSummary
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "RenderStats with valid data",
			args: args{
				stats: StatsSummary{
					StartDate: startDate,
					EndDate:   endDate,
					Days: []DailyStats{
						{
							Date:      startDate,
							TotalTime: time.Duration(30+60) * time.Minute,
							Budgets: []BudgetStats{
								{
									Budget:   budget1,
									Duration: time.Duration(30) * time.Minute,
								},
								{
									Budget:   budget2,
									Duration: time.Duration(60) * time.Minute,
								},
							},
						},
						{
							Date:      startDate.AddDate(0, 0, 1),
							TotalTime: time.Duration(90+120) * time.Minute,
							Budgets: []BudgetStats{
								{
									Budget:   budget1,
									Duration: time.Duration(90) * time.Minute,
								},
								{
									Budget:   budget2,
									Duration: time.Duration(120) * time.Minute,
								},
							},
						},
					},
					Budgets: []BudgetStats{
						{
							Budget:    budget1,
							Duration:  time.Duration(120) * time.Minute,
							Remaining: time.Duration(300-120) * time.Minute,
						},
						{
							Budget:    budget2,
							Duration:  time.Duration(180) * time.Minute,
							Remaining: time.Duration(240-180) * time.Minute,
						},
					},
					TotalPlanned:   time.Duration(300+240) * time.Minute,
					TotalTime:      time.Duration(300) * time.Minute,
					TotalRemaining: time.Duration(240+300-300) * time.Minute,
				},
			},
			want: ",Budget 1,Budget 2,SUM\n" +
				"Planned weekly,05:00:00,04:00:00,09:00:00\n" +
				"01/01/2024,00:30:00,01:00:00,01:30:00\n" +
				"02/01/2024,01:30:00,02:00:00,03:30:00\n" +
				"Total,02:00:00,03:00:00,05:00:00\n" +
				"Remaining,03:00:00,01:00:00,04:00:00\n",
		},
		{
			name: "RenderStats with budget overrides",
			args: args{
				stats: StatsSummary{
					StartDate: startDate,
					EndDate:   endDate,
					Days: []DailyStats{
						{
							Date:      startDate,
							TotalTime: time.Duration(30+60) * time.Minute,
							Budgets: []BudgetStats{
								{
									Budget:   budget1,
									Duration: time.Duration(30) * time.Minute,
								},
								{
									Budget:   budget2,
									Duration: time.Duration(60) * time.Minute,
								},
							},
						},
						{
							Date:      startDate.AddDate(0, 0, 1),
							TotalTime: time.Duration(90+120) * time.Minute,
							Budgets: []BudgetStats{
								{
									Budget:   budget1,
									Duration: time.Duration(90) * time.Minute,
								},
								{
									Budget:   budget2,
									Duration: time.Duration(120) * time.Minute,
								},
							},
						},
					},
					Budgets: []BudgetStats{
						{
							Budget:         budget1,
							BudgetOverride: &budget1Override,
							Duration:       time.Duration(120) * time.Minute,
							Remaining:      time.Duration(300-120) * time.Minute,
						},
						{
							Budget:         budget2,
							BudgetOverride: &budget2Override,
							Duration:       time.Duration(180) * time.Minute,
							Remaining:      time.Duration(240-180) * time.Minute,
						},
					},
					TotalPlanned:   time.Duration(150+360) * time.Minute,
					TotalTime:      time.Duration(300) * time.Minute,
					TotalRemaining: time.Duration(150+360-300) * time.Minute,
				},
			},
			want: ",Budget 1,Budget 2,SUM\n" +
				"Planned weekly,02:30:00,06:00:00,08:30:00\n" +
				"01/01/2024,00:30:00,01:00:00,01:30:00\n" +
				"02/01/2024,01:30:00,02:00:00,03:30:00\n" +
				"Total,02:00:00,03:00:00,05:00:00\n" +
				"Remaining,03:00:00,01:00:00,03:30:00\n",
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &CsvStatsRendererImpl{}
			if got, _ := t.RenderStats(tt.args.stats); got != tt.want {
				t1.Errorf("RenderStats() = %v, want %v", got, tt.want)
			}
		})
	}
}
