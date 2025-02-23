package event

import (
	"encoding/json"
	"errors"
	"github.com/klokku/klokku/internal/rest"
	"github.com/klokku/klokku/pkg/budget"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type EventDTO struct {
	ID        int              `json:"id"`
	Budget    budget.BudgetDTO `json:"budget"`
	StartTime string           `json:"startTime"`
	EndTime   string           `json:"endTime,omitempty"`
	Notes     string           `json:"notes,omitempty"`
}

type EventUpdateRequest struct {
	Status *string `json:"status"`
}

type EventHandler struct {
	eventService EventService
}

func NewEventHandler(eventService EventService) *EventHandler {
	return &EventHandler{eventService}
}

func (e *EventHandler) StartEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Starting new event")

	var startEventRequest struct {
		BudgetId  int    `json:"budgetId"`
		StartTime string `json:"startTime"`
	}

	if err := json.NewDecoder(r.Body).Decode(&startEventRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Invalid request body format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	log.Debug("New event request: ", startEventRequest)

	if len(startEventRequest.StartTime) == 0 {
		startEventRequest.StartTime = time.Now().Format(time.RFC3339)
	}

	startTime, err := time.Parse(time.RFC3339, startEventRequest.StartTime)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid start time format",
			Details: "Start time must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	event := &Event{
		Budget: budget.Budget{
			ID: startEventRequest.BudgetId,
		},
		StartTime: startTime,
	}

	storedEvent, err := e.eventService.StartNewEvent(r.Context(), *event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	eventResponse := eventToDTO(storedEvent)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(eventResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (e *EventHandler) GetCurrentEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	event, err := e.eventService.FindCurrentEvent(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if event == nil {
		http.Error(w, "No current event", http.StatusNotFound)
		return
	}
	eventResponse := eventToDTO(*event)
	if err := json.NewEncoder(w).Encode(eventResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (e *EventHandler) FinishCurrentEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	eventUpdateRequest := EventUpdateRequest{}

	if err := json.NewDecoder(r.Body).Decode(&eventUpdateRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Invalid request body format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	if eventUpdateRequest.Status == nil || *eventUpdateRequest.Status != "finished" {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Invalid or missing 'status' in request body",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	finishedEvents, err := e.eventService.FinishCurrentEvent(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if finishedEvents == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	response := make([]EventDTO, 0, len(finishedEvents))
	for _, calendarEvent := range finishedEvents {
		response = append(response, eventToDTO(calendarEvent))
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (e *EventHandler) ModifyCurrentEventStartTime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Modifying current event start time")
	var modifyEventStartTimeRequest struct {
		StartTime string `json:"startTime"`
	}
	if err := json.NewDecoder(r.Body).Decode(&modifyEventStartTimeRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Invalid request body format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	startTime, err := time.Parse(time.RFC3339, modifyEventStartTimeRequest.StartTime)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid startTime format",
			Details: "Start time must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	modifiedEvent, err := e.eventService.ModifyCurrentEventStartTime(r.Context(), startTime)
	if err != nil {
		if errors.Is(err, ErrNoCurrentEvent) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(eventToDTO(modifiedEvent)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (e *EventHandler) DeleteCurrentEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Deleting current event")

	event, err := e.eventService.DeleteCurrentEvent(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if event == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(event); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (e *EventHandler) GetLast5Events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Getting last 5 events")

	events, err := e.eventService.GetLastPreviousEvents(r.Context(), 5)
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
	log.Trace("Events returned: ", eventsDTO)
}

func eventToDTO(event Event) EventDTO {
	var endTime string
	if !event.EndTime.IsZero() {
		endTime = event.EndTime.Format(time.RFC3339)
	}
	return EventDTO{
		ID:        event.ID,
		Budget:    budget.BudgetToDTO(event.Budget),
		StartTime: event.StartTime.Format(time.RFC3339),
		EndTime:   endTime,
		Notes:     "",
	}
}
