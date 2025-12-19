package current_event

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/klokku/klokku/internal/rest"
	log "github.com/sirupsen/logrus"
)

type CurrentEventDTO struct {
	PlanItem  PlanItemDTO `json:"planItem"`
	StartTime string      `json:"startTime"`
}

type PlanItemDTO struct {
	BudgetItemId   int    `json:"budgetItemId"`
	Name           string `json:"name"`
	WeeklyDuration int    `json:"weeklyDuration"`
}

type EventUpdateRequest struct {
	Status *string `json:"status"`
}

type EventHandler struct {
	eventService Service
}

func NewEventHandler(eventService Service) *EventHandler {
	return &EventHandler{eventService}
}

func (e *EventHandler) StartEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Trace("Starting new current event")

	var startEventRequest struct {
		BudgetItemId int `json:"budgetItemId"`
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

	log.Debug("New current event request: ", startEventRequest)

	startTime := time.Now()

	event := &CurrentEvent{
		StartTime: startTime,
		PlanItem: PlanItem{
			BudgetItemId: startEventRequest.BudgetItemId,
		},
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
	if event.Id == 0 {
		http.Error(w, "No current event", http.StatusNotFound)
		return
	}
	eventResponse := eventToDTO(event)
	if err := json.NewEncoder(w).Encode(eventResponse); err != nil {
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

func eventToDTO(event CurrentEvent) CurrentEventDTO {
	return CurrentEventDTO{
		PlanItem:  planItemToDTO(event.PlanItem),
		StartTime: event.StartTime.Format(time.RFC3339),
	}
}

func planItemToDTO(planItem PlanItem) PlanItemDTO {
	return PlanItemDTO{
		BudgetItemId:   planItem.BudgetItemId,
		Name:           planItem.Name,
		WeeklyDuration: int(planItem.WeeklyDuration.Seconds()),
	}
}
