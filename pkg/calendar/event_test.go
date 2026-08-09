package calendar

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateNotes(t *testing.T) {
	tests := []struct {
		name    string
		notes   string
		wantErr error
	}{
		{name: "accepts empty notes", notes: ""},
		{name: "accepts maximum ASCII length", notes: strings.Repeat("a", MaxNotesLength)},
		{name: "accepts maximum Unicode character length", notes: strings.Repeat("ą", MaxNotesLength)},
		{name: "rejects notes over maximum length", notes: strings.Repeat("a", MaxNotesLength+1), wantErr: ErrNotesTooLong},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateNotes(test.notes)
			if test.wantErr != nil {
				assert.ErrorIs(t, err, test.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
