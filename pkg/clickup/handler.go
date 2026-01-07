package clickup

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/gorilla/mux"
)

type WorkspaceDTO struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type SpaceDTO struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type FolderDTO struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type TagDTO struct {
	Name string `json:"name"`
}

type ConfigurationDTO struct {
	WorkspaceId           string             `json:"workspaceId"`
	SpaceId               string             `json:"spaceId"`
	FolderId              string             `json:"folderId"`
	OnlyTasksWithPriority bool               `json:"onlyTasksWithPriority"`
	Mappings              []BudgetMappingDTO `json:"mappings"`
}

type BudgetMappingDTO struct {
	ClickUpSpaceId string `json:"clickUpSpaceId"`
	ClickUpTagName string `json:"clickUpTagName"`
	BudgetItemId   int    `json:"budgetItemId"`
	Position       int    `json:"position"`
}

type TaskDTO struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	TimeEstimateSec int    `json:"timeEstimateSec"`
}

type Handler struct {
	service Service
	client  Client
}

func NewHandler(s Service, c Client) *Handler {
	return &Handler{s, c}
}

// ListWorkspaces godoc
// @Summary List ClickUp workspaces
// @Description Get all ClickUp workspaces the user has access to
// @Tags ClickUp
// @Produce json
// @Success 200 {array} WorkspaceDTO
// @Failure 403 {string} string "Unauthorized"
// @Router /api/integrations/clickup/workspace [get]
// @Security XUserId
func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	workspaces, err := h.client.GetAuthorizedWorkspaces(r.Context())
	if err != nil {
		if errors.Is(err, ErrUnathenticated) {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	workspacesDTO := make([]WorkspaceDTO, 0, len(workspaces))
	for _, workspace := range workspaces {
		workspaceDTO := WorkspaceDTO(workspace)
		workspacesDTO = append(workspacesDTO, workspaceDTO)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(workspacesDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ListSpaces godoc
// @Summary List ClickUp spaces
// @Description Get all spaces in a ClickUp workspace
// @Tags ClickUp
// @Produce json
// @Param workspaceId query int true "Workspace ID"
// @Success 200 {array} SpaceDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/integrations/clickup/space [get]
// @Security XUserId
func (h *Handler) ListSpaces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	workspaceId := r.URL.Query().Get("workspaceId")
	if workspaceId == "" {
		http.Error(w, "workspaceId is required", http.StatusBadRequest)
		return
	}

	spaces, err := h.client.GetSpaces(r.Context(), workspaceId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spacesDTO := make([]SpaceDTO, 0, len(spaces))
	for _, space := range spaces {
		spaceDTO := SpaceDTO(space)
		spacesDTO = append(spacesDTO, spaceDTO)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(spacesDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ListTags godoc
// @Summary List ClickUp tags
// @Description Get all tags in a ClickUp space
// @Tags ClickUp
// @Produce json
// @Param spaceId query int true "Space ID"
// @Success 200 {array} TagDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/integrations/clickup/tag [get]
// @Security XUserId
func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	spaceId := r.URL.Query().Get("spaceId")
	if spaceId == "" {
		http.Error(w, "spaceId is required", http.StatusBadRequest)
		return
	}
	tags, err := h.client.GetTags(r.Context(), spaceId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tagsDTO := make([]TagDTO, 0, len(tags))
	for _, tag := range tags {
		tagDTO := TagDTO{
			Name: tag.Name,
		}
		tagsDTO = append(tagsDTO, tagDTO)
	}

	// sort tags by name
	sort.Slice(tagsDTO, func(i, j int) bool {
		return tagsDTO[i].Name < tagsDTO[j].Name
	})

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(tagsDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ListFolders godoc
// @Summary List ClickUp folders
// @Description Get all folders in a ClickUp space
// @Tags ClickUp
// @Produce json
// @Param spaceId query int true "Space ID"
// @Success 200 {array} FolderDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/integrations/clickup/folder [get]
// @Security XUserId
func (h *Handler) ListFolders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	spaceId := r.URL.Query().Get("spaceId")
	if spaceId == "" {
		http.Error(w, "spaceId is required", http.StatusBadRequest)
		return
	}

	folders, err := h.client.GetFolders(r.Context(), spaceId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	foldersDTO := make([]FolderDTO, 0, len(folders))
	for _, folder := range folders {
		folderDTO := FolderDTO(folder)
		foldersDTO = append(foldersDTO, folderDTO)
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(foldersDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// StoreConfiguration godoc
// @Summary Store ClickUp configuration
// @Description Save ClickUp integration configuration and budget mappings
// @Tags ClickUp
// @Accept json
// @Param configuration body ConfigurationDTO true "ClickUp Configuration"
// @Success 200 "OK"
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/integrations/clickup/configuration [put]
// @Security XUserId
func (h *Handler) StoreConfiguration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	budgetPlanIdString := vars["budgetPlanId"]
	if budgetPlanIdString == "" {
		http.Error(w, "budgetPlanId is required", http.StatusBadRequest)
		return
	}
	budgetPlanId, err := strconv.Atoi(budgetPlanIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var configurationDTO ConfigurationDTO
	if err := json.NewDecoder(r.Body).Decode(&configurationDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mappings := make([]BudgetItemMapping, 0, len(configurationDTO.Mappings))
	for _, mappingDTO := range configurationDTO.Mappings {
		mappings = append(mappings, BudgetItemMapping{
			ClickupSpaceId: mappingDTO.ClickUpSpaceId,
			ClickupTagName: mappingDTO.ClickUpTagName,
			BudgetItemId:   mappingDTO.BudgetItemId,
			Position:       mappingDTO.Position,
		})
	}

	configuration := Configuration{
		WorkspaceId:           configurationDTO.WorkspaceId,
		SpaceId:               configurationDTO.SpaceId,
		FolderId:              configurationDTO.FolderId,
		OnlyTasksWithPriority: configurationDTO.OnlyTasksWithPriority,
		Mappings:              mappings,
	}

	err = h.service.StoreConfiguration(r.Context(), budgetPlanId, configuration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetConfiguration godoc
// @Summary Get ClickUp configuration
// @Description Retrieve the current ClickUp integration configuration
// @Tags ClickUp
// @Produce json
// @Success 200 {object} ConfigurationDTO
// @Failure 403 {string} string "User not found"
// @Router /api/integrations/clickup/configuration [get]
// @Security XUserId
func (h *Handler) GetConfiguration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	budgetPlanIdString := vars["budgetPlanId"]
	if budgetPlanIdString == "" {
		http.Error(w, "budgetPlanId is required", http.StatusBadRequest)
		return
	}
	budgetPlanId, err := strconv.Atoi(budgetPlanIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	configuration, err := h.service.GetConfiguration(r.Context(), budgetPlanId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	configurationDTO := ConfigurationDTO{
		WorkspaceId:           configuration.WorkspaceId,
		SpaceId:               configuration.SpaceId,
		FolderId:              configuration.FolderId,
		OnlyTasksWithPriority: configuration.OnlyTasksWithPriority,
		Mappings:              make([]BudgetMappingDTO, 0, len(configuration.Mappings)),
	}
	for _, mapping := range configuration.Mappings {
		configurationDTO.Mappings = append(configurationDTO.Mappings, BudgetMappingDTO{
			ClickUpSpaceId: mapping.ClickupSpaceId,
			ClickUpTagName: mapping.ClickupTagName,
			BudgetItemId:   mapping.BudgetItemId,
			Position:       mapping.Position,
		})
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(configurationDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// DisableIntegration godoc
// @Summary Disable ClickUp integration
// @Description Disconnect and disable the ClickUp integration
// @Tags ClickUp
// @Success 200 "OK"
// @Failure 403 {string} string "User not found"
// @Router /api/integrations/clickup/auth [delete]
// @Security XUserId
func (h *Handler) DisableIntegration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := h.service.DisableIntegration(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetTasks godoc
// @Summary Get ClickUp tasks
// @Description Retrieve ClickUp tasks for a specific budget item
// @Tags ClickUp
// @Produce json
// @Param budgetItemId query int true "Budget Item ID"
// @Success 200 {array} TaskDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/integrations/clickup/tasks [get]
// @Security XUserId
func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	budgetItemIdString := r.URL.Query().Get("budgetItemId")
	if budgetItemIdString == "" {
		http.Error(w, "budgetItemId is required", http.StatusBadRequest)
		return
	}
	budgetItemId, err := strconv.Atoi(budgetItemIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tasks, err := h.service.GetTasksByBudgetItemId(r.Context(), budgetItemId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tasksDTO := make([]TaskDTO, 0, len(tasks))
	for _, task := range tasks {
		taskDTO := TaskDTO{
			Id:              task.Id,
			Name:            task.Name,
			TimeEstimateSec: task.TimeEstimateMs / 1000,
		}
		tasksDTO = append(tasksDTO, taskDTO)
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(tasksDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteBudgetPlanConfiguration godoc
// @Summary Delete ClickUp configuration for budget plan
// @Description Remove ClickUp integration configuration for a specific budget plan
// @Tags ClickUp
// @Param budgetPlanId path int true "Budget Plan ID"
// @Success 204 "No Content"
// @Failure 400 {string} string "Bad Request"
// @Router /api/integrations/clickup/configuration/{budgetPlanId} [delete]
// @Security XUserId
func (h *Handler) DeleteBudgetPlanConfiguration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	budgetPlanIdString := vars["budgetPlanId"]
	if budgetPlanIdString == "" {
		http.Error(w, "budgetPlanId is required", http.StatusBadRequest)
		return
	}
	budgetPlanId, err := strconv.Atoi(budgetPlanIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.service.DeleteBudgetPlanConfiguration(r.Context(), budgetPlanId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
