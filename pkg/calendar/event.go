package calendar

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

const MaxNotesLength = 10_000

var ErrNotesTooLong = errors.New("notes exceed maximum length")

type Event struct {
	UID       string
	Summary   string
	StartTime time.Time
	EndTime   time.Time
	Notes     string
	Metadata  EventMetadata
}

type EventPatch struct {
	StartTime    *time.Time
	EndTime      *time.Time
	Notes        *string
	BudgetItemId *int
}

type EventMetadata struct {
	BudgetItemId int `json:"budgetItemId"`
}

func ValidateNotes(notes string) error {
	length := utf8.RuneCountInString(notes)
	if length > MaxNotesLength {
		return fmt.Errorf("%w: got %d characters, maximum is %d", ErrNotesTooLong, length, MaxNotesLength)
	}
	return nil
}
