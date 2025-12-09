package stats

import (
	"bytes"
	"encoding/csv"
	log "github.com/sirupsen/logrus"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CsvStatsRendererImpl struct {
}

func NewCsvStatsTransformer() *CsvStatsRendererImpl {
	return &CsvStatsRendererImpl{}
}

func (t *CsvStatsRendererImpl) RenderStats(stats StatsSummary) (string, error) {

	budgetsWeeklyTimes := make([]string, 0, len(stats.Budgets)+2)
	budgetsWeeklyTimes = append(budgetsWeeklyTimes, "Planned weekly")
	budgetNames := make([]string, 0, len(stats.Budgets)+2)
	budgetNames = append(budgetNames, "")
	for _, budgetStats := range stats.Budgets {
		budgetNames = append(budgetNames, budgetStats.Budget.Name)
		weeklyTime := budgetStats.Budget.WeeklyDuration
		if budgetStats.BudgetOverride != nil {
			weeklyTime = budgetStats.BudgetOverride.WeeklyTime
		}
		budgetsWeeklyTimes = append(budgetsWeeklyTimes, durationToString(weeklyTime))
	}
	budgetsWeeklyTimes = append(budgetsWeeklyTimes, durationToString(stats.TotalPlanned))

	statsByDay := make([][]string, 0, len(stats.Days))
	for _, dailyStats := range stats.Days {
		dayStats := getStatsForDay(dailyStats, budgetNames[1:])
		statsByDay = append(statsByDay, dayStats)
	}

	statsByBudget := make([]string, 0, len(stats.Budgets)+2)
	statsByBudget = append(statsByBudget, "Total")
	for _, budgetStats := range stats.Budgets {
		statsByBudget = append(statsByBudget, durationToString(budgetStats.Duration))
	}
	statsByBudget = append(statsByBudget, durationToString(stats.TotalTime))

	remainingTime := make([]string, 0, len(stats.Budgets)+2)
	remainingTime = append(remainingTime, "Remaining")
	for _, budgetStats := range stats.Budgets {
		remainingTime = append(remainingTime, durationToString(budgetStats.Remaining))
	}
	remainingTime = append(remainingTime, durationToString(stats.TotalRemaining))

	budgetNames = append(budgetNames, "SUM")
	data := make([][]string, 0, 2+len(statsByDay)+2)
	data = append(data, budgetNames, budgetsWeeklyTimes)
	data = append(data, statsByDay...)
	data = append(data, statsByBudget, remainingTime)

	var b bytes.Buffer
	writer := csv.NewWriter(&b)
	for _, row := range data {
		err := writer.Write(row)
		if err != nil {
			log.Errorf("Error writing to csv: %v", err)
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Errorf("Error writing to csv: %v", err)
		return "", err
	}

	return b.String(), nil
}

func getStatsForDay(dailyStats DailyStats, budgetNames []string) []string {
	sort.Slice(dailyStats.Budgets, func(i, j int) bool {
		return dailyStats.Budgets[i].Budget.Name < dailyStats.Budgets[j].Budget.Name
	})
	dayStats := make([]string, 0, len(dailyStats.Budgets)+2)
	dayStats = append(dayStats, dailyStats.Date.Format("02/01/2006"))
	for _, name := range budgetNames {
		idx, found := slices.BinarySearchFunc(dailyStats.Budgets, name, func(budgetStats BudgetStats, name string) int {
			return strings.Compare(budgetStats.Budget.Name, name)
		})
		if found {
			dayStats = append(dayStats, durationToString(dailyStats.Budgets[idx].Duration))
		} else {
			dayStats = append(dayStats, "00:00:00")
		}
	}
	dayStats = append(dayStats, durationToString(dailyStats.TotalTime))
	return dayStats
}

func durationToString(duration time.Duration) string {
	hours := strconv.Itoa(int(duration.Hours()))
	if len(hours) == 1 {
		hours = "0" + hours
	}
	minutes := strconv.Itoa(int(duration.Minutes()) % 60)
	if len(minutes) == 1 {
		minutes = "0" + minutes
	}
	seconds := strconv.Itoa(int(duration.Seconds()) % 60)
	if len(seconds) == 1 {
		seconds = "0" + seconds
	}
	return hours + ":" + minutes + ":" + seconds
}
