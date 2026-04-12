package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetReport(planID int, from, to string) (*ReportDTO, error) {
	var report ReportDTO
	path := fmt.Sprintf("/api/budgetplan/%d/report", planID)
	if from != "" && to != "" {
		path += fmt.Sprintf("?from=%s&to=%s", url.QueryEscape(from), url.QueryEscape(to))
	}
	if err := c.Get(path, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func (c *Client) GetItemReport(planID, itemID int, from, to string) (*ItemDetailReportDTO, error) {
	var report ItemDetailReportDTO
	path := fmt.Sprintf("/api/budgetplan/%d/report/item/%d", planID, itemID)
	if from != "" && to != "" {
		path += fmt.Sprintf("?from=%s&to=%s", url.QueryEscape(from), url.QueryEscape(to))
	}
	if err := c.Get(path, &report); err != nil {
		return nil, err
	}
	return &report, nil
}
