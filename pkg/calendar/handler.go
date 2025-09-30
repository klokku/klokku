package calendar

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/rest"
	"github.com/klokku/klokku/pkg/budget"
)

type Handler struct {
	calendar        *Service
	budgetsProvider BudgetsProviderFunc
}

type BudgetsProviderFunc func(ctx context.Context, includeInactive bool) ([]budget.Budget, error)

type EventDTO struct {
	UID       string    `json:"uid"`
	Summary   string    `json:"summary"`
	StartTime time.Time `json:"start"`
	EndTime   time.Time `json:"end"`
	BudgetId  int       `json:"budgetId"`
}

func NewHandler(s *Service, bp BudgetsProviderFunc) *Handler {
	return &Handler{s, bp}
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
		dto, err := eventToDTO(e)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		dtos = append(dtos, dto)
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
	event, err := dtoToEvent(eventDTO)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	budgets, err := h.budgetsProvider(r.Context(), false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var budgetInfo *budget.Budget
	for _, b := range budgets {
		if b.ID == event.Metadata.BudgetId {
			budgetInfo = &b
			break
		}
	}
	if budgetInfo == nil {
		http.Error(w, "Invalid budget id", http.StatusBadRequest)
		return
	}
	event.Summary = budgetInfo.Name

	addedEvent, err := h.calendar.AddStickyEvent(r.Context(), event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	addedEventDTO, err := eventToDTO(*addedEvent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(addedEventDTO); err != nil {
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
	event, err := dtoToEvent(eventDTO)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	budgets, err := h.budgetsProvider(r.Context(), false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var budgetInfo *budget.Budget
	for _, b := range budgets {
		if b.ID == event.Metadata.BudgetId {
			budgetInfo = &b
			break
		}
	}
	if budgetInfo == nil {
		http.Error(w, "Invalid budget id", http.StatusBadRequest)
		return
	}
	event.Summary = budgetInfo.Name

	modifiedEvent, err := h.calendar.ModifyStickyEvent(r.Context(), event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	modifiedEventDTO, err := eventToDTO(*modifiedEvent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(modifiedEventDTO); err != nil {
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

func eventToDTO(e Event) (EventDTO, error) {
	return EventDTO{
		UID:       e.UID,
		Summary:   e.Summary,
		StartTime: e.StartTime,
		EndTime:   e.EndTime,
		BudgetId:  e.Metadata.BudgetId,
	}, nil
}

func dtoToEvent(e EventDTO) (Event, error) {
	return Event{
		UID:       e.UID,
		Summary:   e.Summary,
		StartTime: e.StartTime,
		EndTime:   e.EndTime,
		Metadata:  EventMetadata{BudgetId: e.BudgetId},
	}, nil
}
