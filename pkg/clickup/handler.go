package clickup

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
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
	WorkspaceId int                `json:"workspaceId"`
	SpaceId     int                `json:"spaceId"`
	FolderId    int                `json:"folderId"`
	Mappings    []BudgetMappingDTO `json:"mappings"`
}

type BudgetMappingDTO struct {
	ClickUpSpaceId int    `json:"clickUpSpaceId"`
	ClickUpTagName string `json:"clickUpTagName"`
	BudgetId       int    `json:"budgetId"`
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

func (h *Handler) ListSpaces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	workspaceIdString := r.URL.Query().Get("workspaceId")
	if workspaceIdString == "" {
		http.Error(w, "workspaceId is required", http.StatusBadRequest)
		return
	}
	workspaceId, err := strconv.Atoi(workspaceIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	spaceIdString := r.URL.Query().Get("spaceId")
	if spaceIdString == "" {
		http.Error(w, "spaceId is required", http.StatusBadRequest)
		return
	}
	spaceId, err := strconv.Atoi(spaceIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

func (h *Handler) ListFolders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	spaceIdString := r.URL.Query().Get("spaceId")
	if spaceIdString == "" {
		http.Error(w, "spaceId is required", http.StatusBadRequest)
	}
	spaceId, err := strconv.Atoi(spaceIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

func (h *Handler) StoreConfiguration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var configurationDTO ConfigurationDTO
	if err := json.NewDecoder(r.Body).Decode(&configurationDTO); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	mappings := make([]BudgetMapping, 0, len(configurationDTO.Mappings))
	for _, mappingDTO := range configurationDTO.Mappings {
		mappings = append(mappings, BudgetMapping{
			ClickupSpaceId: mappingDTO.ClickUpSpaceId,
			ClickupTagName: mappingDTO.ClickUpTagName,
			BudgetItemId:   mappingDTO.BudgetId,
			Position:       mappingDTO.Position,
		})
	}

	configuration := Configuration{
		WorkspaceId: configurationDTO.WorkspaceId,
		SpaceId:     configurationDTO.SpaceId,
		FolderId:    configurationDTO.FolderId,
		Mappings:    mappings,
	}

	err := h.service.StoreConfiguration(r.Context(), configuration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetConfiguration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	configuration, err := h.service.GetConfiguration(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	configurationDTO := ConfigurationDTO{
		WorkspaceId: configuration.WorkspaceId,
		SpaceId:     configuration.SpaceId,
		FolderId:    configuration.FolderId,
		Mappings:    make([]BudgetMappingDTO, 0, len(configuration.Mappings)),
	}
	for _, mapping := range configuration.Mappings {
		configurationDTO.Mappings = append(configurationDTO.Mappings, BudgetMappingDTO{
			ClickUpSpaceId: mapping.ClickupSpaceId,
			ClickUpTagName: mapping.ClickupTagName,
			BudgetId:       mapping.BudgetItemId,
			Position:       mapping.Position,
		})
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(configurationDTO); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) DisableIntegration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := h.service.DisableIntegration(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	budgetIdString := r.URL.Query().Get("budgetId")
	if budgetIdString == "" {
		http.Error(w, "budgetId is required", http.StatusBadRequest)
		return
	}
	budgetId, err := strconv.Atoi(budgetIdString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tasks, err := h.service.GetTasksByBudgetId(r.Context(), budgetId)
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
