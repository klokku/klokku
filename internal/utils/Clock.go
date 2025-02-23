package utils

import "time"

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (s SystemClock) Now() time.Time {
	return time.Now()
}

type MockClock struct {
	FixedNow time.Time
}

func (m *MockClock) Now() time.Time {
	return m.FixedNow
}

func (m *MockClock) SetNow(now time.Time) {
	m.FixedNow = now
}
