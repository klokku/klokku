package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserService_CreateUser_EventCalendarType(t *testing.T) {
	tests := []struct {
		name         string
		calendarType EventCalendarType
		want         EventCalendarType
		wantErr      error
	}{
		{name: "defaults empty value", want: KlokkuCalendar},
		{name: "rejects invalid value", calendarType: "invalid", wantErr: ErrUserDataInvalid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewUserService(NewStubUserRepository())
			created, err := service.CreateUser(context.Background(), validUser(test.calendarType))

			require.ErrorIs(t, err, test.wantErr)
			if test.wantErr == nil {
				require.Equal(t, test.want, created.Settings.EventCalendarType)
			}
		})
	}
}

func TestUserService_UpdateUser_EventCalendarType(t *testing.T) {
	tests := []struct {
		name         string
		calendarType EventCalendarType
		want         EventCalendarType
		wantErr      error
	}{
		{name: "defaults empty value", want: KlokkuCalendar},
		{name: "rejects invalid value", calendarType: "invalid", wantErr: ErrUserDataInvalid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := NewStubUserRepository()
			repo.data[1] = validUser(GoogleCalendar)
			service := NewUserService(repo)
			ctx := WithUser(context.Background(), User{Id: 1})

			updated, err := service.UpdateUser(ctx, validUser(test.calendarType))

			require.ErrorIs(t, err, test.wantErr)
			if test.wantErr == nil {
				require.Equal(t, test.want, updated.Settings.EventCalendarType)
			}
		})
	}
}

func TestUpdateUser_InvalidEventCalendarType(t *testing.T) {
	handler := NewHandler(NewUserService(NewStubUserRepository()))
	body, err := json.Marshal(userToDTO(&User{
		Username:    "test-user",
		DisplayName: "Test User",
		Settings: Settings{
			EventCalendarType: "invalid",
		},
	}))
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPut, "/api/user/current", bytes.NewReader(body))
	req = req.WithContext(WithUser(req.Context(), User{Id: 1}))
	response := httptest.NewRecorder()

	handler.UpdateUser(response, req)

	require.Equal(t, http.StatusBadRequest, response.Code)
}

func validUser(calendarType EventCalendarType) User {
	return User{
		Uid:         "test-uid",
		Username:    "test-user",
		DisplayName: "Test User",
		Settings: Settings{
			EventCalendarType: calendarType,
		},
	}
}
