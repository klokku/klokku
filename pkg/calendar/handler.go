package calendar

import (
	"encoding/json"
	"net/http"
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

func (h *Handler) GetLast5Events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Getting last 5 events")

	events, err := h.calendar.GetLastEvents(r.Context(), 5)
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
