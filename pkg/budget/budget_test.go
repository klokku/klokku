package budget

import (
	"testing"
	"time"
)

func TestBudget_IsActiveBetween(t *testing.T) {
	type fields struct {
		ID                int
		Name              string
		WeeklyTime        time.Duration
		WeeklyOccurrences int
		StartDate         time.Time
		EndDate           time.Time
		Icon              string
		Position          int
	}
	type args struct {
		startDate time.Time
		endDate   time.Time
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// Case 1: Budget with no dates (infinite duration)
		{
			name: "Budget with no dates (always active)",
			fields: fields{
				StartDate: time.Time{}, // Zero time
				EndDate:   time.Time{}, // Zero time
			},
			args: args{
				startDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 2: Exact match between budget dates and query dates
		{
			name: "Exact match between budget dates and query dates",
			fields: fields{
				StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 3: Budget with no start date (active until end date)
		{
			name: "Budget with no start date (active until end date)",
			fields: fields{
				StartDate: time.Time{}, // Zero time
				EndDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 4: Budget with no end date (active from start date)
		{
			name: "Budget with no end date (active from start date)",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Time{}, // Zero time
			},
			args: args{
				startDate: time.Date(2021, 1, 20, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 5: Query period completely before budget period
		{
			name: "Query period completely before budget period",
			fields: fields{
				StartDate: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 2, 28, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: false,
		},

		// Case 6: Query period completely after budget period
		{
			name: "Query period completely after budget period",
			fields: fields{
				StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 2, 28, 0, 0, 0, 0, time.UTC),
			},
			want: false,
		},

		// Case 7: Query period overlaps with the beginning of budget period
		{
			name: "Query period overlaps with the beginning of budget period",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 2, 15, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 8: Query period overlaps with the end of budget period
		{
			name: "Query period overlaps with the end of budget period",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 2, 15, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 2, 28, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 9: Query period completely contains budget period
		{
			name: "Query period completely contains budget period",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 20, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 10: Budget period completely contains query period
		{
			name: "Budget period completely contains query period",
			fields: fields{
				StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 1, 10, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 20, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 11: Single day overlap at the start
		{
			name: "Single day overlap at the start",
			fields: fields{
				StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 12: Single day overlap at the end
		{
			name: "Single day overlap at the end",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 13: No overlap - budget ends exactly when query starts
		{
			name: "No overlap - budget ends exactly when query starts",
			fields: fields{
				StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 1, 15, 0, 0, 0, 1, time.UTC), // One nanosecond after budget end
				endDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: false,
		},

		// Case 14: No overlap - query ends exactly when budget starts
		{
			name: "No overlap - query ends exactly when budget starts",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				startDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2021, 1, 14, 23, 59, 59, 999999999, time.UTC), // One nanosecond before budget start
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Budget{
				ID:                tt.fields.ID,
				Name:              tt.fields.Name,
				WeeklyTime:        tt.fields.WeeklyTime,
				WeeklyOccurrences: tt.fields.WeeklyOccurrences,
				StartDate:         tt.fields.StartDate,
				EndDate:           tt.fields.EndDate,
				Icon:              tt.fields.Icon,
				Position:          tt.fields.Position,
			}
			if got := b.IsActiveBetween(tt.args.startDate, tt.args.endDate); got != tt.want {
				t.Errorf("IsActiveBetween() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBudget_IsActiveOn(t *testing.T) {
	type fields struct {
		ID                int
		Name              string
		WeeklyTime        time.Duration
		WeeklyOccurrences int
		StartDate         time.Time
		EndDate           time.Time
		Icon              string
		Position          int
	}
	type args struct {
		date time.Time
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// Case 1: Budget with no dates (infinite duration)
		{
			name: "Budget with no start and end dates (always active)",
			fields: fields{
				StartDate: time.Time{}, // Zero time
				EndDate:   time.Time{}, // Zero time
			},
			args: args{
				date: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 2: Date is exactly on start date
		{
			name: "Date is exactly on start date",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				date: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 3: Date is exactly on end date
		{
			name: "Date is exactly on end date",
			fields: fields{
				StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				date: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 4: Date is between start and end dates
		{
			name: "Date is between start and end dates",
			fields: fields{
				StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				date: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 5: Date is before start date
		{
			name: "Date is before start date",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: false,
		},

		// Case 6: Date is after end date
		{
			name: "Date is after end date",
			fields: fields{
				StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				date: time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: false,
		},

		// Case 7: Budget with no start date but with end date
		{
			name: "Budget with no start date but with end date",
			fields: fields{
				StartDate: time.Time{}, // Zero time
				EndDate:   time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 8: Budget with no start date, date is after end date
		{
			name: "Budget with no start date, date is after end date",
			fields: fields{
				StartDate: time.Time{}, // Zero time
				EndDate:   time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				date: time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: false,
		},

		// Case 9: Budget with no end date but with start date
		{
			name: "Budget with no end date but with start date",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Time{}, // Zero time
			},
			args: args{
				date: time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			want: true,
		},

		// Case 10: Budget with no end date, date is before start date
		{
			name: "Budget with no end date, date is before start date",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Time{}, // Zero time
			},
			args: args{
				date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: false,
		},

		// Case 11: Date is 1 nanosecond after start date
		{
			name: "Date is 1 nanosecond after start date",
			fields: fields{
				StartDate: time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				date: time.Date(2021, 1, 15, 0, 0, 0, 1, time.UTC),
			},
			want: true,
		},

		// Case 12: Date is 1 nanosecond before end date
		{
			name: "Date is 1 nanosecond before end date",
			fields: fields{
				StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
			},
			args: args{
				date: time.Date(2021, 1, 14, 23, 59, 59, 999999999, time.UTC),
			},
			want: true,
		},

		// Case 13: Date is exactly on end date with zero time
		{
			name: "Date is exactly on end date with zero time",
			fields: fields{
				StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Time{}, // Zero time
			},
			args: args{
				date: time.Time{}, // Zero time
			},
			want: false, // StartDate.Before(date) is false, should return false
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Budget{
				ID:                tt.fields.ID,
				Name:              tt.fields.Name,
				WeeklyTime:        tt.fields.WeeklyTime,
				WeeklyOccurrences: tt.fields.WeeklyOccurrences,
				StartDate:         tt.fields.StartDate,
				EndDate:           tt.fields.EndDate,
				Icon:              tt.fields.Icon,
				Position:          tt.fields.Position,
			}
			if got := b.IsActiveOn(tt.args.date); got != tt.want {
				t.Errorf("IsActiveOn() = %v, want %v", got, tt.want)
			}
		})
	}
}
