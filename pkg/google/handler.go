package google

import (
	"encoding/json"
	"errors"
	"net/http"
)

type CalendarItemDto struct {
	Id      string `json:"id"`
	Summary string `json:"summary"`
}

type Handler struct {
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{s}
}

func (h *Handler) ListCalendars(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	calendars, err := h.service.ListCalendars(r.Context())
	if err != nil {
		if errors.Is(err, ErrUnathenticated) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	calendarItems := make([]CalendarItemDto, 0, len(calendars))
	for _, c := range calendars {
		calendarItems = append(calendarItems, toCalendarItemDto(c))
	}

	if err := json.NewEncoder(w).Encode(calendarItems); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func toCalendarItemDto(ci CalendarItem) CalendarItemDto {
	return CalendarItemDto{
		Id:      ci.ID,
		Summary: ci.Summary,
	}
}
