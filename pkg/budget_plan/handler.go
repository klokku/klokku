package budget_plan

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type BudgetPlanDTO struct {
	Id        int       `json:"id"`
	Name      string    `json:"name"`
	IsCurrent bool      `json:"iscurrent"`
	Items     []ItemDTO `json:"items,omitempty"`
}

type ItemDTO struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	WeeklyDuration    int    `json:"weeklyDuration"`
	WeeklyOccurrences int    `json:"weeklyOccurrences,omitempty"`
	Icon              string `json:"icon,omitempty"`
	Color             string `json:"color,omitempty"`
}

type Handler struct {
	service Service
}

func NewBudgetPlanHandler(service Service) *Handler {
	return &Handler{service}
}

// ListPlans godoc
// @Summary List all budget plans
// @Description Get a list of all budget plans for the current user
// @Tags BudgetPlan
// @Produce json
// @Success 200 {array} BudgetPlanDTO
// @Failure 403 {string} string "User not found"
// @Router /api/budgetplan [get]
// @Security XUserId
func (handler *Handler) ListPlans(w http.ResponseWriter, r *http.Request) {
	log.Debug("Listing budget plans")
	w.Header().Set("Content-Type", "application/json")
	plans, err := handler.service.ListPlans(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	plansDTO := make([]BudgetPlanDTO, 0, len(plans))
	for _, plan := range plans {
		plansDTO = append(plansDTO, PlanToDTO(plan))
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(plansDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CreatePlan godoc
// @Summary Create a new budget plan
// @Description Create a new budget plan with the provided details
// @Tags BudgetPlan
// @Accept json
// @Produce json
// @Param plan body BudgetPlanDTO true "Budget Plan"
// @Success 201 {object} BudgetPlanDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/budgetplan [post]
// @Security XUserId
func (handler *Handler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	log.Debug("Creating new budget plan")
	w.Header().Set("Content-Type", "application/json")
	var planDTO BudgetPlanDTO
	if err := json.NewDecoder(r.Body).Decode(&planDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	plan, err := handler.service.CreatePlan(r.Context(), DTOToPlan(planDTO))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	planDTO = PlanToDTO(plan)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(planDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UpdatePlan godoc
// @Summary Update an existing budget plan
// @Description Update a budget plan by ID
// @Tags BudgetPlan
// @Accept json
// @Produce json
// @Param planId path int true "Budget Plan ID"
// @Param plan body BudgetPlanDTO true "Budget Plan"
// @Success 200 {object} BudgetPlanDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Failure 404 {string} string "Plan Not Found"
// @Router /api/budgetplan/{planId} [put]
// @Security XUserId
func (handler *Handler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	log.Debug("Updating budget plan")
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	planIdString := vars["planId"]
	planId, err := strconv.Atoi(planIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var planDTO BudgetPlanDTO
	if err := json.NewDecoder(r.Body).Decode(&planDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if planDTO.Id == 0 || planDTO.Id != planId {
		http.Error(w, "Invalid plan id in request body", http.StatusBadRequest)
		return
	}
	plan := DTOToPlan(planDTO)
	updatedPlan, err := handler.service.UpdatePlan(r.Context(), plan)
	if err != nil {
		if errors.Is(err, ErrPlanNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	updatedPlanDTO := PlanToDTO(updatedPlan)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedPlanDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeletePlan godoc
// @Summary Delete a budget plan
// @Description Delete a budget plan by ID
// @Tags BudgetPlan
// @Param planId path int true "Budget Plan ID"
// @Success 204 "No Content"
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Failure 404 {string} string "Plan Not Found"
// @Router /api/budgetplan/{planId} [delete]
// @Security XUserId
func (handler *Handler) DeletePlan(w http.ResponseWriter, r *http.Request) {
	log.Debug("Deleting budget plan")
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	planIdString := vars["planId"]
	planId, err := strconv.Atoi(planIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	deleted, err := handler.service.DeletePlan(r.Context(), planId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !deleted {
		http.Error(w, "plan not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RegisterItem godoc
// @Summary Add a new budget item to a plan
// @Description Register a new budget item within a specific budget plan
// @Tags BudgetItem
// @Accept json
// @Produce json
// @Param planId path int true "Budget Plan ID"
// @Param item body ItemDTO true "Budget Item"
// @Success 201 {object} ItemDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/budgetplan/{planId}/item [post]
// @Security XUserId
func (handler *Handler) RegisterItem(w http.ResponseWriter, r *http.Request) {
	log.Debug("Registering new budget item")
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	planIdString := vars["planId"]
	planId, err := strconv.Atoi(planIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var itemDTO ItemDTO
	if err := json.NewDecoder(r.Body).Decode(&itemDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item := DTOToItem(planId, itemDTO)

	createdItem, err := handler.service.CreateItem(r.Context(), item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	createdItemDto := ItemToDTO(createdItem)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(createdItemDto); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetPlan godoc
// @Summary Get a budget plan by ID
// @Description Retrieve a specific budget plan with all its items
// @Tags BudgetPlan
// @Produce json
// @Param planId path int true "Budget Plan ID"
// @Success 200 {object} BudgetPlanDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Failure 404 {string} string "Plan Not Found"
// @Router /api/budgetplan/{planId} [get]
// @Security XUserId
func (handler *Handler) GetPlan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	planIdString := vars["planId"]
	planId, err := strconv.Atoi(planIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	plan, err := handler.service.GetPlan(r.Context(), planId)
	if err != nil {
		if errors.Is(err, ErrPlanNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	planDto := PlanToDTO(plan)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(planDto); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UpdateItem godoc
// @Summary Update a budget item
// @Description Update an existing budget item within a plan
// @Tags BudgetItem
// @Accept json
// @Produce json
// @Param planId path int true "Budget Plan ID"
// @Param itemId path int true "Budget Item ID"
// @Param item body ItemDTO true "Budget Item"
// @Success 200 {object} ItemDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/budgetplan/{planId}/item/{itemId} [put]
// @Security XUserId
func (handler *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	planIdString := vars["planId"]
	planId, err := strconv.Atoi(planIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	itemIdString := vars["itemId"]
	itemId, err := strconv.Atoi(itemIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var itemDTO ItemDTO
	if err := json.NewDecoder(r.Body).Decode(&itemDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if itemDTO.ID == 0 || itemDTO.ID != itemId {
		http.Error(w, "Invalid item id in request body", http.StatusBadRequest)
		return
	}

	item := DTOToItem(planId, itemDTO)
	updatedItem, err := handler.service.UpdateItem(r.Context(), item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	updatedItemDTO := ItemToDTO(updatedItem)

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(updatedItemDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteItem godoc
// @Summary Delete a budget item
// @Description Remove a budget item from a plan
// @Tags BudgetItem
// @Param planId path int true "Budget Plan ID"
// @Param itemId path int true "Budget Item ID"
// @Success 204 "No Content"
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Failure 404 {string} string "Item Not Found"
// @Router /api/budgetplan/{planId}/item/{itemId} [delete]
// @Security XUserId
func (handler *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	itemIdString := vars["itemId"]
	itemId, err := strconv.Atoi(itemIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ok, err := handler.service.DeleteItem(r.Context(), itemId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	// Return 204 No Content for successful deletion with no response body
	w.WriteHeader(http.StatusNoContent)
}

// SetItemPosition godoc
// @Summary Set position of a budget item
// @Description Move a budget item to a specific position in the list
// @Tags BudgetItem
// @Accept json
// @Param planId path int true "Budget Plan ID"
// @Param itemId path int true "Budget Item ID"
// @Param position body object{id=int,precedingId=int} true "Position details"
// @Success 200 "OK"
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Failure 404 {string} string "Item Not Found"
// @Router /api/budgetplan/{planId}/item/{itemId}/position [put]
// @Security XUserId
func (handler *Handler) SetItemPosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	planIdString := vars["planId"]
	planId, err := strconv.Atoi(planIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	itemIdString := vars["itemId"]
	itemId, err := strconv.Atoi(itemIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var setPositionDTO struct {
		ID          int `json:"id"`
		PrecedingId int `json:"precedingId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&setPositionDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ok, err := handler.service.MoveItemAfter(r.Context(), planId, itemId, setPositionDTO.PrecedingId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "BudgetItem not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func PlanToDTO(plan BudgetPlan) BudgetPlanDTO {
	itemsDto := make([]ItemDTO, 0, len(plan.Items))
	for _, item := range plan.Items {
		itemsDto = append(itemsDto, ItemToDTO(item))
	}
	return BudgetPlanDTO{
		Id:        plan.Id,
		Name:      plan.Name,
		Items:     itemsDto,
		IsCurrent: plan.IsCurrent,
	}
}

func DTOToPlan(planDTO BudgetPlanDTO) BudgetPlan {
	items := make([]BudgetItem, 0, len(planDTO.Items))
	for _, itemDTO := range planDTO.Items {
		items = append(items, DTOToItem(planDTO.Id, itemDTO))
	}
	return BudgetPlan{
		Id:    planDTO.Id,
		Name:  planDTO.Name,
		Items: items,
	}
}

func ItemToDTO(item BudgetItem) ItemDTO {
	return ItemDTO{
		ID:                item.Id,
		Name:              item.Name,
		WeeklyDuration:    int(item.WeeklyDuration.Seconds()),
		WeeklyOccurrences: item.WeeklyOccurrences,
		Icon:              item.Icon,
		Color:             item.Color,
	}
}

func DTOToItem(planId int, itemDTO ItemDTO) BudgetItem {

	return BudgetItem{
		Id:                itemDTO.ID,
		PlanId:            planId,
		Name:              itemDTO.Name,
		WeeklyDuration:    time.Duration(itemDTO.WeeklyDuration) * time.Second,
		WeeklyOccurrences: itemDTO.WeeklyOccurrences,
		Icon:              itemDTO.Icon,
		Color:             itemDTO.Color,
	}
}
