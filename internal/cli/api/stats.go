package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetWeeklyStats(date string) (*WeeklyStatsSummaryDTO, error) {
	var stats WeeklyStatsSummaryDTO
	if err := c.Get("/api/stats/weekly?date="+url.QueryEscape(date), &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (c *Client) GetItemHistory(from, to string, budgetItemID int) (*PlanItemHistoryStatsDTO, error) {
	var stats PlanItemHistoryStatsDTO
	path := fmt.Sprintf("/api/stats/item-history?from=%s&to=%s&budgetItemId=%d",
		url.QueryEscape(from), url.QueryEscape(to), budgetItemID)
	if err := c.Get(path, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}
