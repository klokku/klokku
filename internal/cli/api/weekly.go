package api

import (
	"fmt"
	"net/url"
)

func (c *Client) GetWeeklyPlan(date string) (*WeeklyPlanDTO, error) {
	var plan WeeklyPlanDTO
	if err := c.Get("/api/weeklyplan?date="+url.QueryEscape(date), &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func (c *Client) ResetWeeklyPlan(date string) (*WeeklyPlanDTO, error) {
	var plan WeeklyPlanDTO
	req, err := c.newRequest("DELETE", "/api/weeklyplan?date="+url.QueryEscape(date), nil)
	if err != nil {
		return nil, err
	}
	if err := c.do(req, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func (c *Client) SetOffWeek(date string, isOff bool) (*WeeklyPlanDTO, error) {
	body, err := jsonBody(SetOffWeekRequest{IsOffWeek: isOff})
	if err != nil {
		return nil, err
	}
	var plan WeeklyPlanDTO
	if err := c.Put("/api/weeklyplan/off-week?date="+url.QueryEscape(date), body, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func (c *Client) UpdateWeeklyItem(date string, r UpdateWeeklyItemRequest) (*WeeklyPlanItemDTO, error) {
	body, err := jsonBody(r)
	if err != nil {
		return nil, err
	}
	var item WeeklyPlanItemDTO
	if err := c.Put("/api/weeklyplan/item?date="+url.QueryEscape(date), body, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *Client) ResetWeeklyItem(itemID int) (*WeeklyPlanItemDTO, error) {
	var item WeeklyPlanItemDTO
	req, err := c.newRequest("DELETE", fmt.Sprintf("/api/weeklyplan/item/%d", itemID), nil)
	if err != nil {
		return nil, err
	}
	if err := c.do(req, &item); err != nil {
		return nil, err
	}
	return &item, nil
}
