package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/klokku/klokku/pkg/user"
	"github.com/stretchr/testify/require"
)

type memoryRepository struct {
	configurations map[int]*Configuration
}

func (r *memoryRepository) Find(ctx context.Context, userId int) (*Configuration, error) {
	return r.configurations[userId], nil
}

func (r *memoryRepository) Store(ctx context.Context, userId int, configuration *Configuration) error {
	r.configurations[userId] = configuration
	return nil
}

func request(handler http.HandlerFunc, method, body string, userId int) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/user/current/dashboard", strings.NewReader(body))
	req = req.WithContext(user.WithUser(req.Context(), user.User{Id: userId}))
	response := httptest.NewRecorder()
	handler(response, req)
	return response
}

func TestHandler_ConfigurationRoundTrip(t *testing.T) {
	handler := NewHandler(NewService(&memoryRepository{configurations: make(map[int]*Configuration)}))
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
		{"duplicate cards", `{"cards":[{"id":"time-today","visible":true},{"id":"time-today","visible":false}]}`},
		{"missing visibility", `{"cards":[{"id":"time-today"}]}`},
		{"null visibility", `{"cards":[{"id":"time-today","visible":null}]}`},
		{"missing cards", `{}`},
		{"null cards", `{"cards":null}`},
		{"null configuration", `null`},
		{"empty id", `{"cards":[{"id":"","visible":true}]}`},
		{"tracking card", `{"cards":[{"id":"tracking-now","visible":false}]}`},
		{"trailing JSON", `{"cards":[]} {}`},
		{"invalid id", `{"cards":[{"id":"bad id","visible":true}]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHandler(NewService(&memoryRepository{configurations: make(map[int]*Configuration)}))
			body := `{"cards":[{"id":"time-today","visible":true}]}`
			require.Equal(t, http.StatusOK, request(handler.SaveConfiguration, http.MethodPut, body, 1).Code)
			require.Equal(t, http.StatusBadRequest, request(handler.SaveConfiguration, http.MethodPut, test.body, 1).Code)
			require.JSONEq(t, body, request(handler.GetConfiguration, http.MethodGet, "", 1).Body.String())
		})
	}
}
