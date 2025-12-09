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
	service BudgetPlanService
}

func NewBudgetPlanHandler(service BudgetPlanService) *Handler {
	return &Handler{service}
}

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
	ok, err := handler.service.UpdateItem(r.Context(), item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(itemDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

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
