package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/require"
)

func request(handler http.HandlerFunc, method, body string, userId int) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/user/current/dashboard", strings.NewReader(body))
	req = req.WithContext(user.WithUser(req.Context(), user.User{Id: userId}))
	response := httptest.NewRecorder()
	handler(response, req)
	return response
}

func TestHandler_ConfigurationRoundTrip(t *testing.T) {
	handler := NewHandler(NewService(newRepositoryStub()))
	require.JSONEq(t, `null`, request(handler.GetConfiguration, http.MethodGet, "", 1).Body.String())
	body := `{"cards":[{"id":"time-today","visible":false},{"id":"future-card","visible":true}]}`
	response := request(handler.SaveConfiguration, http.MethodPut, body, 1)
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, body, response.Body.String())
	require.JSONEq(t, body, request(handler.GetConfiguration, http.MethodGet, "", 1).Body.String())
	require.JSONEq(t, `null`, request(handler.GetConfiguration, http.MethodGet, "", 2).Body.String())
	require.Equal(t, http.StatusNoContent, request(handler.ResetConfiguration, http.MethodDelete, "", 1).Code)
	require.JSONEq(t, `null`, request(handler.GetConfiguration, http.MethodGet, "", 1).Body.String())
}

func TestHandler_InvalidConfigurationDoesNotReplacePreferences(t *testing.T) {
	tests := []struct{ name, body string }{
		{name: "service validation error maps to bad request", body: `{"cards":[{"id":"time-today","visible":true},{"id":"time-today","visible":false}]}`},
		{name: "missing visibility", body: `{"cards":[{"id":"time-today"}]}`},
		{name: "null visibility", body: `{"cards":[{"id":"time-today","visible":null}]}`},
		{name: "missing cards", body: `{}`},
		{name: "null cards", body: `{"cards":null}`},
		{name: "null configuration", body: `null`},
		{name: "trailing JSON", body: `{"cards":[]} {}`},
		{name: "malformed JSON", body: `{"cards":`},
		{name: "unknown field", body: `{"cards":[],"userId":2}`},
		{name: "oversized body", body: strings.Repeat(" ", 16*1024) + `{"cards":[]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandler(NewService(newRepositoryStub()))
			body := `{"cards":[{"id":"time-today","visible":true}]}`
			require.Equal(t, http.StatusOK, request(handler.SaveConfiguration, http.MethodPut, body, 1).Code)
			require.Equal(t, http.StatusBadRequest, request(handler.SaveConfiguration, http.MethodPut, test.body, 1).Code)
			require.JSONEq(t, body, request(handler.GetConfiguration, http.MethodGet, "", 1).Body.String())
		})
	}
}
