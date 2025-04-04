package budget

import (
	"encoding/json"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

type BudgetDTO struct {
	ID                int          `json:"id"`
	Name              string       `json:"name"`
	WeeklyTime        int          `json:"weeklyTime"`
	WeeklyOccurrences int          `json:"weeklyOccurrences,omitempty"`
	Icon              string       `json:"icon,omitempty"`
	Status            BudgetStatus `json:"status,omitempty"`
}

type BudgetHandler struct {
	budgetService BudgetService
}

func NewBudgetHandler(budgetService BudgetService) *BudgetHandler {
	return &BudgetHandler{budgetService}
}

func (handler *BudgetHandler) Register(w http.ResponseWriter, r *http.Request) {
	log.Debug("Registering new budget")
	w.Header().Set("Content-Type", "application/json")

	var budgetDTO BudgetDTO
	if err := json.NewDecoder(r.Body).Decode(&budgetDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	budget := DTOToBudget(budgetDTO)

	createdBudget, err := handler.budgetService.Create(r.Context(), budget)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	createdBudgetDto := BudgetToDTO(createdBudget)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(createdBudgetDto); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (handler *BudgetHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	includeInactive := r.URL.Query().Has("includeInactive")

	budgets, err := handler.budgetService.GetAll(r.Context(), includeInactive)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	budgetsDTO := make([]BudgetDTO, 0, len(budgets))
	for _, budget := range budgets {
		budgetDTO := BudgetToDTO(budget)
		budgetsDTO = append(budgetsDTO, budgetDTO)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(budgetsDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (handler *BudgetHandler) Update(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	budgetIdString := vars["id"]
	budgetId, err := strconv.ParseInt(budgetIdString, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var budgetDTO BudgetDTO
	if err := json.NewDecoder(r.Body).Decode(&budgetDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if budgetDTO.ID == 0 || budgetDTO.ID != int(budgetId) {
		http.Error(w, "Invalid budget id in request body", http.StatusBadRequest)
		return
	}

	budget := DTOToBudget(budgetDTO)
	ok, err := handler.budgetService.Update(r.Context(), budget)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Budget not found", http.StatusNotFound)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(budgetDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (handler *BudgetHandler) SetPosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	budgetIdString := vars["id"]
	budgetId, err := strconv.ParseInt(budgetIdString, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	var setPositionDTO struct {
		ID          int64 `json:"id"`
		PrecedingId int64 `json:"precedingId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&setPositionDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	ok, err := handler.budgetService.MoveAfter(r.Context(), budgetId, setPositionDTO.PrecedingId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Budget not found", http.StatusNotFound)
	}
	w.WriteHeader(http.StatusOK)
}

func BudgetToDTO(budget Budget) BudgetDTO {
	return BudgetDTO{
		ID:                budget.ID,
		Name:              budget.Name,
		WeeklyTime:        int(budget.WeeklyTime.Seconds()),
		WeeklyOccurrences: budget.WeeklyOccurrences,
		Icon:              budget.Icon,
		Status:            budget.Status,
	}
}

func DTOToBudget(budgetDTO BudgetDTO) Budget {
	return Budget{
		ID:                budgetDTO.ID,
		Name:              budgetDTO.Name,
		WeeklyTime:        time.Duration(budgetDTO.WeeklyTime) * time.Second,
		WeeklyOccurrences: budgetDTO.WeeklyOccurrences,
		Icon:              budgetDTO.Icon,
		Status:            budgetDTO.Status,
	}
}
