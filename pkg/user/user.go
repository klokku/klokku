package user

import "time"

type User struct {
	Id          int
	Uid         string
	Username    string
	DisplayName string
	PhotoUrl    string
	Settings    Settings
}

type EventCalendarType string

const (
	KlokkuCalendar EventCalendarType = "klokku"
	GoogleCalendar EventCalendarType = "google"
)

// IsValid reports whether the calendar type is one of the supported values.
func (t EventCalendarType) IsValid() bool {
	switch t {
	case KlokkuCalendar, GoogleCalendar:
		return true
	}
	return false
}

type Settings struct {
	Timezone          string
	WeekFirstDay      time.Weekday
	EventCalendarType EventCalendarType
	GoogleCalendar    GoogleCalendarSettings
	IgnoreShortEvents bool
}

type GoogleCalendarSettings struct {
	CalendarId string
}
