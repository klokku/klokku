package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_AuthHeaders_Token(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UserDTO{UID: "test"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "my-token", "")
	_, err := client.GetCurrentUser()
	require.NoError(t, err)
	assert.Equal(t, "Bearer my-token", gotAuth)
}

func TestClient_AuthHeaders_UserID(t *testing.T) {
	var gotUserID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = r.Header.Get("X-User-Id")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UserDTO{UID: "test"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "", "user-123")
	_, err := client.GetCurrentUser()
	require.NoError(t, err)
	assert.Equal(t, "user-123", gotUserID)
}

func TestClient_ErrorResponse_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "bad request", Details: "missing field"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", "")
	_, err := client.GetCurrentUser()
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 400, apiErr.StatusCode)
	assert.Equal(t, "bad request", apiErr.Message)
	assert.Equal(t, "missing field", apiErr.Details)
}

func TestClient_ErrorResponse_PlainText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "user not found", http.StatusForbidden)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", "")
	_, err := client.GetCurrentUser()
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 403, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "user not found")
}

func TestClient_NoContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", "")
	err := client.Delete("/api/test")
	require.NoError(t, err)
}

func TestClient_ListBudgetPlans(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/budgetplan", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]BudgetPlanDTO{
			{ID: 1, Name: "Work", IsCurrent: true},
			{ID: 2, Name: "Side project"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", "")
	plans, err := client.ListBudgetPlans()
	require.NoError(t, err)
	assert.Len(t, plans, 2)
	assert.Equal(t, "Work", plans[0].Name)
	assert.True(t, plans[0].IsCurrent)
}

func TestClient_CreateBudgetItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/budgetplan/1/item", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var item BudgetItemDTO
		json.NewDecoder(r.Body).Decode(&item)
		assert.Equal(t, "Coding", item.Name)
		assert.Equal(t, 28800, item.WeeklyDuration)

		item.ID = 42
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", "")
	item, err := client.CreateBudgetItem(1, BudgetItemDTO{
		Name:           "Coding",
		WeeklyDuration: 28800,
	})
	require.NoError(t, err)
	assert.Equal(t, 42, item.ID)
}

func TestClient_StartEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/event", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var req StartEventRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, 5, req.BudgetItemID)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(CurrentEventDTO{
			PlanItem:  PlanItemDTO{BudgetItemID: 5, Name: "Work"},
			StartTime: "2026-04-05T09:00:00Z",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", "")
	event, err := client.StartEvent(5, "Work", 28800)
	require.NoError(t, err)
	assert.Equal(t, "Work", event.PlanItem.Name)
}

func TestNewClientNoAuth(t *testing.T) {
	var gotAuth, gotUserID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUserID = r.Header.Get("X-User-Id")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(WebhookTriggerResponse{Success: true})
	}))
	defer srv.Close()

	client := NewClientNoAuth(srv.URL)
	resp, err := client.TriggerWebhook("abc")
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Empty(t, gotAuth)
	assert.Empty(t, gotUserID)
}
