package budget_override

import (
	"encoding/json"
	"github.com/gorilla/mux"
	rest "github.com/klokku/klokku/internal/rest"
	"net/http"
	"strconv"
	"time"
)

type BudgetOverrideDTO struct {
	ID         int     `json:"id"`
	BudgetID   int     `json:"budgetId"`
	StartDate  string  `json:"startDate"`
	WeeklyTime int32   `json:"weeklyTime"`
	Notes      *string `json:"notes"`
}

type BudgetOverrideHandler struct {
	service BudgetOverrideService
}

func NewBudgetOverrideHandler(service BudgetOverrideService) *BudgetOverrideHandler {
	return &BudgetOverrideHandler{
		service: service,
	}
}

func (h *BudgetOverrideHandler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var overrideRequest BudgetOverrideDTO

	if err := json.NewDecoder(r.Body).Decode(&overrideRequest); err != nil {
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

	startDate, err := time.Parse(time.RFC3339, overrideRequest.StartDate)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid startDate format",
			Details: "Start date must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	override := BudgetOverride{
		ID:         0,
		BudgetID:   overrideRequest.BudgetID,
		StartDate:  startDate,
		WeeklyTime: time.Duration(overrideRequest.WeeklyTime) * time.Second,
		Notes:      overrideRequest.Notes,
	}

	storedOverride, err := h.service.Store(r.Context(), override)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(overrideToDTO(storedOverride)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *BudgetOverrideHandler) GetOverrides(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	weekStartDateString := r.URL.Query().Get("startDate")
	weekStartDate, err := time.Parse(time.RFC3339, weekStartDateString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Incorrect startDate format",
			Details: "Start date must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	overrides, err := h.service.GetAllForWeek(r.Context(), weekStartDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	overridesDTO := make([]BudgetOverrideDTO, 0, len(overrides))
	for _, override := range overrides {
		overrideDTO := overrideToDTO(override)
		overridesDTO = append(overridesDTO, overrideDTO)
	}

	if err := json.NewEncoder(w).Encode(overridesDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func overrideToDTO(override BudgetOverride) BudgetOverrideDTO {
	return BudgetOverrideDTO{
		ID:         override.ID,
		BudgetID:   override.BudgetID,
		StartDate:  override.StartDate.Format(time.RFC3339),
		WeeklyTime: int32(override.WeeklyTime.Seconds()),
		Notes:      override.Notes,
	}
}

func (h *BudgetOverrideHandler) Update(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var overrideRequest BudgetOverrideDTO

	if err := json.NewDecoder(r.Body).Decode(&overrideRequest); err != nil {
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

	startDate, err := time.Parse(time.RFC3339, overrideRequest.StartDate)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid startDate format",
			Details: "Start date must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	override := BudgetOverride{
		ID:         overrideRequest.ID,
		BudgetID:   overrideRequest.BudgetID,
		StartDate:  startDate,
		WeeklyTime: time.Duration(overrideRequest.WeeklyTime) * time.Second,
		Notes:      overrideRequest.Notes,
	}

	err = h.service.Update(r.Context(), override)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(overrideToDTO(override)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *BudgetOverrideHandler) Delete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	overrideIdString := vars["overrideId"]
	overrideId, err := strconv.ParseInt(overrideIdString, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid overrideId format",
			Details: "Parameter overrideId must be a number",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	err = h.service.Delete(r.Context(), int(overrideId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
