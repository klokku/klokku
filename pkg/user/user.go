package user

import "time"

type User struct {
	Id          int
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

type Settings struct {
	Timezone          string
	WeekFirstDay      time.Weekday
	EventCalendarType EventCalendarType
	GoogleCalendar    GoogleCalendarSettings
}

type GoogleCalendarSettings struct {
	CalendarId string
}
