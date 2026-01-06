package weekly_plan

import (
	"reflect"
	"testing"
	"time"
)

var location, _ = time.LoadLocation("Europe/Warsaw")

func TestWeekNumberEqual(t *testing.T) {
	tests := []struct {
		name   string
		left   WeekNumber
		right  WeekNumber
		expect bool
	}{
		{"same year and week", WeekNumber{Year: 2025, Week: 3}, WeekNumber{Year: 2025, Week: 3}, true},
		{"different week", WeekNumber{Year: 2025, Week: 3}, WeekNumber{Year: 2025, Week: 4}, false},
		{"different year", WeekNumber{Year: 2024, Week: 52}, WeekNumber{Year: 2025, Week: 52}, false},
		{"zero values equal", WeekNumber{}, WeekNumber{}, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.left.Equal(tt.right); got != tt.expect {
				t.Fatalf("Equal(%+v, %+v) = %v, want %v", tt.left, tt.right, got, tt.expect)
			}
		})
	}
}

func TestWeekNumberBefore(t *testing.T) {
	tests := []struct {
		name   string
		left   WeekNumber
		right  WeekNumber
		expect bool
	}{
		{"same week", WeekNumber{Year: 2025, Week: 3}, WeekNumber{Year: 2025, Week: 3}, false},
		{"earlier week same year", WeekNumber{Year: 2025, Week: 2}, WeekNumber{Year: 2025, Week: 3}, true},
		{"later week same year", WeekNumber{Year: 2025, Week: 4}, WeekNumber{Year: 2025, Week: 3}, false},
		{"earlier year", WeekNumber{Year: 2024, Week: 52}, WeekNumber{Year: 2025, Week: 1}, true},
		{"later year", WeekNumber{Year: 2026, Week: 1}, WeekNumber{Year: 2025, Week: 52}, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.left.Before(tt.right); got != tt.expect {
				t.Fatalf("Before(%+v, %+v) = %v, want %v", tt.left, tt.right, got, tt.expect)
			}
		})
	}
}

func TestWeekNumberAfter(t *testing.T) {
	tests := []struct {
		name   string
		left   WeekNumber
		right  WeekNumber
		expect bool
	}{
		{"same week", WeekNumber{Year: 2025, Week: 3}, WeekNumber{Year: 2025, Week: 3}, false},
		{"later week same year", WeekNumber{Year: 2025, Week: 4}, WeekNumber{Year: 2025, Week: 3}, true},
		{"earlier week same year", WeekNumber{Year: 2025, Week: 2}, WeekNumber{Year: 2025, Week: 3}, false},
		{"later year", WeekNumber{Year: 2026, Week: 1}, WeekNumber{Year: 2025, Week: 52}, true},
		{"earlier year", WeekNumber{Year: 2024, Week: 52}, WeekNumber{Year: 2025, Week: 1}, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.left.After(tt.right); got != tt.expect {
				t.Fatalf("After(%+v, %+v) = %v, want %v", tt.left, tt.right, got, tt.expect)
			}
		})
	}
}

func TestWeekNumberFromDate(t *testing.T) {
	type args struct {
		date         time.Time
		weekStartDay time.Weekday
	}

	sundayStart := time.Date(2024, time.December, 29, 0, 0, 0, 0, time.UTC)
	sundayYear, sundayWeek := sundayStart.ISOWeek()

	saturdayStart := time.Date(2024, time.December, 28, 0, 0, 0, 0, time.UTC)
	saturdayYear, saturdayWeek := saturdayStart.ISOWeek()

	tests := []struct {
		name string
		args args
		want WeekNumber
	}{
		{
			name: "default week start day is Monday",
			args: args{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Monday},
			want: WeekNumber{Year: 2025, Week: 1},
		},
		{
			name: "week start day is Sunday - include previous Sunday",
			args: args{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Sunday},
			want: WeekNumber{Year: sundayYear, Week: sundayWeek},
		},
		{
			name: "week start day is Saturday - include previous Saturday",
			args: args{time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC), time.Saturday},
			want: WeekNumber{Year: saturdayYear, Week: saturdayWeek},
		},
		{
			name: "invalid week start day defaults to Monday",
			args: args{time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), time.Weekday(42)},
			want: WeekNumber{Year: 2025, Week: 5},
		},
		{
			name: "first week of the year is the week containing January 4th",
			args: args{time.Date(2025, 12, 29, 0, 0, 0, 0, location), time.Monday},
			want: WeekNumber{Year: 2026, Week: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WeekNumberFromDate(tt.args.date, tt.args.weekStartDay); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WeekNumberFromDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
