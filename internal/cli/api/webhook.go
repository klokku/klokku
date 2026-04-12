package api

import (
	"fmt"
	"net/url"
)

func (c *Client) ListWebhooks(webhookType string) ([]WebhookDTO, error) {
	var webhooks []WebhookDTO
	path := "/api/webhook"
	if webhookType != "" {
		path += "?type=" + url.QueryEscape(webhookType)
	}
	if err := c.Get(path, &webhooks); err != nil {
		return nil, err
	}
	return webhooks, nil
}

func (c *Client) CreateWebhook(req CreateWebhookRequest) (*WebhookDTO, error) {
	body, err := jsonBody(req)
	if err != nil {
		return nil, err
	}
	var webhook WebhookDTO
	if err := c.Post("/api/webhook", body, &webhook); err != nil {
		return nil, err
	}
	return &webhook, nil
}

func (c *Client) DeleteWebhook(id int) error {
	return c.Delete(fmt.Sprintf("/api/webhook/%d", id))
}

// TriggerWebhook calls the webhook endpoint (no auth required).
func (c *Client) TriggerWebhook(token string) (*WebhookTriggerResponse, error) {
	var resp WebhookTriggerResponse
	if err := c.Post("/api/webhook/"+url.PathEscape(token), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
