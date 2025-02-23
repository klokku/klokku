package stats

type StatsRenderer interface {
	RenderStats(stats StatsSummary) (string, error)
}
