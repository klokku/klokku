package calendar

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/rest"
	log "github.com/sirupsen/logrus"
)

type Handler struct {
	calendar *Service
}

type EventDTO struct {
	UID          string    `json:"uid"`
	Summary      string    `json:"summary"`
	StartTime    time.Time `json:"start"`
	EndTime      time.Time `json:"end"`
	BudgetItemId int       `json:"budgetItemId"`
}

func NewHandler(s *Service) *Handler {
	return &Handler{s}
}

// GetEvents godoc
// @Summary Get calendar events
// @Description Retrieve calendar events within a date range
// @Tags Calendar
// @Produce json
// @Param from query string true "Start date in RFC3339 format"
// @Param to query string true "End date in RFC3339 format"
// @Success 200 {array} EventDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid date format"
// @Failure 403 {string} string "User not found"
// @Router /api/calendar/event [get]
// @Security XUserId
func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	fromString := r.URL.Query().Get("from")
	toString := r.URL.Query().Get("to")
	from, err := time.Parse(time.RFC3339, fromString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid from (date) format",
			Details: "'from' must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	to, err := time.Parse(time.RFC3339, toString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid to (date) format",
			Details: "'to' must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	events, err := h.calendar.GetEvents(r.Context(), from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var dtos = make([]EventDTO, 0, len(events))
	for _, e := range events {
		dtos = append(dtos, eventToDTO(e))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(dtos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CreateEvent godoc
// @Summary Create a calendar event
// @Description Add a new event to the calendar
// @Tags Calendar
// @Accept json
// @Produce json
// @Param event body EventDTO true "Calendar Event"
// @Success 201 {array} EventDTO "Array of created events (may include recurring instances)"
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/calendar/event [post]
// @Security XUserId
func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var eventDTO EventDTO
	if err := json.NewDecoder(r.Body).Decode(&eventDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	addedEvents, err := h.calendar.AddStickyEvent(r.Context(), dtoToEvent(eventDTO))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var eventDTOs []EventDTO
	for _, e := range addedEvents {
		eventDTOs = append(eventDTOs, eventToDTO(e))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(eventDTOs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UpdateEvent godoc
// @Summary Update a calendar event
// @Description Modify an existing calendar event
// @Tags Calendar
// @Accept json
// @Produce json
// @Param eventUid path string true "Event UID"
// @Param event body EventDTO true "Updated Calendar Event"
// @Success 200 {array} EventDTO "Array of modified events"
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/calendar/event/{eventUid} [put]
// @Security XUserId
func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	var eventDTO EventDTO
	if err := json.NewDecoder(r.Body).Decode(&eventDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	modifiedEvents, err := h.calendar.ModifyStickyEvent(r.Context(), dtoToEvent(eventDTO))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var eventDTOs []EventDTO
	for _, e := range modifiedEvents {
		eventDTOs = append(eventDTOs, eventToDTO(e))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(eventDTOs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteEvent godoc
// @Summary Delete a calendar event
// @Description Remove a calendar event by UID
// @Tags Calendar
// @Param eventUid path string true "Event UID"
// @Success 204 "No Content"
// @Failure 403 {string} string "User not found"
// @Router /api/calendar/event/{eventUid} [delete]
// @Security XUserId
func (h *Handler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	eventUidString := vars["eventUid"]
	err := h.calendar.DeleteEvent(r.Context(), eventUidString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func eventToDTO(e Event) EventDTO {
	return EventDTO{
		UID:          e.UID,
		Summary:      e.Summary,
		StartTime:    e.StartTime,
		EndTime:      e.EndTime,
		BudgetItemId: e.Metadata.BudgetItemId,
	}
}

func dtoToEvent(e EventDTO) Event {
	return Event{
		UID:       e.UID,
		Summary:   e.Summary,
		StartTime: e.StartTime,
		EndTime:   e.EndTime,
		Metadata:  EventMetadata{BudgetItemId: e.BudgetItemId},
	}
}

// GetLastEvents godoc
// @Summary Get recent calendar events
// @Description Retrieve the most recent calendar events. Specify the number using the 'last' query parameter (e.g., last=5)
// @Tags Calendar
// @Produce json
// @Param last query int true "Number of recent events to retrieve" default(5)
// @Success 200 {array} EventDTO
// @Failure 403 {string} string "User not found"
// @Router /api/calendar/event/recent [get]
// @Security XUserId
func (h *Handler) GetLastEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	lastString := vars["last"]
	last, err := strconv.Atoi(lastString)
	if err != nil || last < 1 {
		log.Warnf("Invalid last parameter: %s. Using default value 5", lastString)
		last = 5
	}

	log.Tracef("Getting last %d events", last)

	events, err := h.calendar.GetLastEvents(r.Context(), last)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	eventsDTO := make([]EventDTO, 0, len(events))
	for _, event := range events {
		eventsDTO = append(eventsDTO, eventToDTO(event))
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(eventsDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Tracef("Events returned: %d", len(eventsDTO))
}
