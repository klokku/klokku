package api

import (
	"fmt"
	"net/url"
)

// --- Current Event ---

func (c *Client) StartEvent(budgetItemID int, name string, weeklyDuration int) (*CurrentEventDTO, error) {
	body, err := jsonBody(StartEventRequest{BudgetItemID: budgetItemID, Name: name, WeeklyDuration: weeklyDuration})
	if err != nil {
		return nil, err
	}
	var event CurrentEventDTO
	if err := c.Post("/api/event", body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (c *Client) GetCurrentEvent() (*CurrentEventDTO, error) {
	var event CurrentEventDTO
	if err := c.Get("/api/event/current", &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (c *Client) AdjustCurrentEventStart(startTime string) (*CurrentEventDTO, error) {
	body, err := jsonBody(AdjustStartRequest{StartTime: startTime})
	if err != nil {
		return nil, err
	}
	var event CurrentEventDTO
	if err := c.Patch("/api/event/current/start", body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// --- Calendar Events ---

func (c *Client) ListCalendarEvents(from, to string) ([]CalendarEventDTO, error) {
	var events []CalendarEventDTO
	path := fmt.Sprintf("/api/calendar/event?from=%s&to=%s", url.QueryEscape(from), url.QueryEscape(to))
	if err := c.Get(path, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *Client) GetRecentEvents(last int) ([]CalendarEventDTO, error) {
	var events []CalendarEventDTO
	if err := c.Get(fmt.Sprintf("/api/calendar/event/recent?last=%d", last), &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *Client) CreateCalendarEvent(event CalendarEventDTO) ([]CalendarEventDTO, error) {
	body, err := jsonBody(event)
	if err != nil {
		return nil, err
	}
	var events []CalendarEventDTO
	if err := c.Post("/api/calendar/event", body, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *Client) UpdateCalendarEvent(eventUID string, event CalendarEventDTO) ([]CalendarEventDTO, error) {
	body, err := jsonBody(event)
	if err != nil {
		return nil, err
	}
	var events []CalendarEventDTO
	if err := c.Put("/api/calendar/event/"+url.PathEscape(eventUID), body, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *Client) DeleteCalendarEvent(eventUID string) error {
	return c.Delete("/api/calendar/event/" + url.PathEscape(eventUID))
}
