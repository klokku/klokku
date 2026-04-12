package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an HTTP client for the Klokku REST API.
type Client struct {
	BaseURL    string
	Token      string // Bearer token for managed mode
	UserID     string // X-User-Id for self-hosted mode
	HTTPClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURL, token, userID string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		UserID:  userID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError represents an error response from the server.
type APIError struct {
	StatusCode int
	Message    string
	Details    string
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("API error %d: %s (%s)", e.StatusCode, e.Message, e.Details)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	} else if c.UserID != "" {
		req.Header.Set("X-User-Id", c.UserID)
	}

	return req, nil
}

func (c *Client) do(req *http.Request, result any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseErrorResponse(resp)
	}

	// No content
	if resp.StatusCode == http.StatusNoContent || result == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{StatusCode: resp.StatusCode, Message: "failed to read error response"}
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    errResp.Error,
			Details:    errResp.Details,
		}
	}

	// Fallback for non-JSON error responses
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	return &APIError{StatusCode: resp.StatusCode, Message: msg}
}

// Get performs a GET request and decodes the response into result.
func (c *Client) Get(path string, result any) error {
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body io.Reader, result any) error {
	req, err := c.newRequest(http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(path string, body io.Reader, result any) error {
	req, err := c.newRequest(http.MethodPut, path, body)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

// Patch performs a PATCH request with a JSON body.
func (c *Client) Patch(path string, body io.Reader, result any) error {
	req, err := c.newRequest(http.MethodPatch, path, body)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) error {
	req, err := c.newRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// jsonBody marshals v to JSON and returns a reader. Returns an error if marshaling fails.
func jsonBody(v any) (*bytes.Reader, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}
	return bytes.NewReader(data), nil
}

// NewClientNoAuth creates a client without authentication (for webhook trigger).
func NewClientNoAuth(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}
