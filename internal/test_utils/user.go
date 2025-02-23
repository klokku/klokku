package test_utils

import (
	"context"
	"github.com/klokku/klokku/pkg/user"
	"time"
)

type TestUserProvider struct{}

func (p TestUserProvider) GetCurrentUser(ctx context.Context) (user.User, error) {
	return user.User{
		Id:          123,
		Username:    "test_user",
		DisplayName: "Test User",
		PhotoUrl:    "",
		Settings: user.Settings{
			Timezone:          "Europe/Warsaw",
			WeekFirstDay:      time.Monday,
			EventCalendarType: user.KlokkuCalendar,
			GoogleCalendar:    user.GoogleCalendarSettings{},
		},
	}, nil
}
