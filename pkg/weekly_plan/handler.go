package weekly_plan

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	rest "github.com/klokku/klokku/internal/rest"
)

type WeeklyPlanItemDTO struct {
	Id                int    `json:"id"`
	BudgetItemId      int    `json:"budgetItemId"`
	Name              string `json:"name"`
	WeeklyDuration    int    `json:"weeklyDuration"`
	WeeklyOccurrences int    `json:"weeklyOccurrences"`
	Icon              string `json:"icon,omitempty"`
	Color             string `json:"color,omitempty"`
	Notes             string `json:"notes"`
	Position          int    `json:"position"`
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GetItems godoc
// @Summary Get weekly plan items
// @Description Retrieve all items for a specific week
// @Tags WeeklyPlan
// @Produce json
// @Param date query string true "Date in RFC3339 format (can be any day of the week)"
// @Success 200 {array} WeeklyPlanItemDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid date format"
// @Failure 403 {string} string "User not found"
// @Router /api/weeklyplan [get]
// @Security XUserId
func (h *Handler) GetItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Can be any day of the given week
	weekDateString := r.URL.Query().Get("date")
	weekDate, err := time.Parse(time.RFC3339, weekDateString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Incorrect date format",
			Details: "Date must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
	}
	items, err := h.service.GetItemsForWeek(r.Context(), weekDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	itemsDTO := make([]WeeklyPlanItemDTO, 0, len(items))
	for _, item := range items {
		itemsDTO = append(itemsDTO, WeeklyPlanItemToDTO(item))
	}
	if err := json.NewEncoder(w).Encode(itemsDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UpdateItem godoc
// @Summary Update a weekly plan item
// @Description Update the duration and notes for a weekly plan item
// @Tags WeeklyPlan
// @Accept json
// @Produce json
// @Param date query string true "Date in RFC3339 format (can be any day of the week)"
// @Param item body object{id=int,budgetItemId=int,weeklyDuration=int,notes=string} true "Item update details"
// @Success 200 {object} WeeklyPlanItemDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid request"
// @Failure 403 {string} string "User not found"
// @Failure 404 {string} string "Item Not Found"
// @Router /api/weeklyplan/item [put]
// @Security XUserId
func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Can be any day of the given week
	weekDateString := r.URL.Query().Get("date")
	weekDate, err := time.Parse(time.RFC3339, weekDateString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Incorrect date format",
			Details: "Date must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
	}

	var updateItemDTO struct {
		Id           int    `json:"id"`
		BudgetItemId int    `json:"budgetItemId"`
		Duration     int    `json:"weeklyDuration"`
		Notes        string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateItemDTO); err != nil {
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
	if updateItemDTO.Id == 0 && updateItemDTO.BudgetItemId == 0 {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error: "Id or budgetItemId must be provided",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
	}

	duration := time.Duration(updateItemDTO.Duration) * time.Second

	updatedItem, err := h.service.UpdateItem(r.Context(), weekDate, updateItemDTO.Id, updateItemDTO.BudgetItemId, duration, updateItemDTO.Notes)
	if err != nil {
		if errors.Is(err, ErrWeeklyPlanItemNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(WeeklyPlanItemToDTO(updatedItem)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ResetItem godoc
// @Summary Reset a weekly plan item
// @Description Reset a weekly plan item to its original budget plan values
// @Tags WeeklyPlan
// @Produce json
// @Param itemId path int true "Weekly Plan Item ID"
// @Success 200 {object} WeeklyPlanItemDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid itemId"
// @Failure 403 {string} string "User not found"
// @Router /api/weeklyplan/item/{itemId} [delete]
// @Security XUserId
func (h *Handler) ResetItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	itemIdString := vars["itemId"]
	itemId, err := strconv.ParseInt(itemIdString, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid itemId format",
			Details: "Parameter itemId must be a number",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	itemAfterReset, err := h.service.ResetWeekItemToBudgetPlanItem(r.Context(), int(itemId))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(WeeklyPlanItemToDTO(itemAfterReset)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ResetWeek godoc
// @Summary Reset entire week
// @Description Reset all weekly plan items to budget plan values for a specific week
// @Tags WeeklyPlan
// @Produce json
// @Param date query string true "Date in RFC3339 format (can be any day of the week)"
// @Success 200 {array} WeeklyPlanItemDTO
// @Failure 400 {object} rest.ErrorResponse "Invalid date format"
// @Failure 403 {string} string "User not found"
// @Router /api/weeklyplan [delete]
// @Security XUserId
func (h *Handler) ResetWeek(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Can be any day of the given week
	weekDateString := r.URL.Query().Get("date")
	weekDate, err := time.Parse(time.RFC3339, weekDateString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Incorrect date format",
			Details: "Date must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
	}
	itemsAfterReset, err := h.service.ResetWeekItemsToBudgetPlan(r.Context(), weekDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var itemsDTO []WeeklyPlanItemDTO
	for _, item := range itemsAfterReset {
		itemsDTO = append(itemsDTO, WeeklyPlanItemToDTO(item))
	}
	if err := json.NewEncoder(w).Encode(itemsDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func WeeklyPlanItemToDTO(item WeeklyPlanItem) WeeklyPlanItemDTO {
	return WeeklyPlanItemDTO{
		Id:                item.Id,
		BudgetItemId:      item.BudgetItemId,
		Name:              item.Name,
		WeeklyDuration:    int(item.WeeklyDuration.Seconds()),
		WeeklyOccurrences: item.WeeklyOccurrences,
		Icon:              item.Icon,
		Color:             item.Color,
		Notes:             item.Notes,
		Position:          item.Position,
	}
}
