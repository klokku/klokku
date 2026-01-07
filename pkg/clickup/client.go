package clickup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

const (
	baseURL = "https://api.clickup.com/api/v2"
)

var ErrUnathenticated = fmt.Errorf("user is not authenticated with ClickUp")

type Workspace struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Space struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Folder struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type TaskPriority struct {
	Id         string `json:"id"`
	OrderIndex string `json:"orderindex"`
	Priority   string `json:"priority"`
}

type Task struct {
	Id             string       `json:"id"`
	Name           string       `json:"name"`
	TimeEstimateMs int          `json:"time_estimate"`
	Priority       TaskPriority `json:"priority"`
}

type Tag struct {
	Name    string `json:"name"`
	FgColor string `json:"tag_fg"`
	BgColor string `json:"tag_bg"`
}

type Client interface {
	GetAuthorizedWorkspaces(ctx context.Context) ([]Workspace, error)   // /v2/oauth/token
	GetSpaces(ctx context.Context, workspaceId string) ([]Space, error) // /v2/team/{team_id}/space
	GetFolders(ctx context.Context, spaceId string) ([]Folder, error)   // /v2/space/{space_id}/folder
	GetFilteredTeamTasks(ctx context.Context, workspaceId string, spaceId string, folderId string, page int, tagName string,
		withPrioritySetOnly bool) ([]Task, error) // /v2/team/{team_Id}/task
	GetTags(ctx context.Context, spaceId string) ([]Tag, error) // /v2/space/{space_id}/tag
}

type ClientImpl struct {
	auth *ClickUpAuth
}

func NewClient(auth *ClickUpAuth) *ClientImpl {
	return &ClientImpl{
		auth: auth,
	}
}

// prepareClickUpClient returns an authenticated HTTP client for the ClickUp API
func (s *ClientImpl) prepareClickUpClient(ctx context.Context) (*http.Client, error) {
	userId, err := user.CurrentId(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	client, err := s.auth.getClient(ctx, userId)
	if err != nil {
		err := fmt.Errorf("unable to retrieve ClickUp auth client: %w", err)
		log.Error(err)
		return nil, err
	}
	if client == nil {
		log.Debug("user is unauthenticated, authentication is required")
		return nil, ErrUnathenticated
	}

	return client, nil
}

// GetAuthorizedWorkspaces retrieves the workspaces the user has access to
func (s *ClientImpl) GetAuthorizedWorkspaces(ctx context.Context) ([]Workspace, error) {
	client, err := s.prepareClickUpClient(ctx)
	if err != nil {
		if errors.Is(err, ErrUnathenticated) {
			return nil, err
		}
		log.Errorf("Failed to prepare ClickUp client: %v", err)
		return nil, err
	}

	// According to ClickUp API docs, the endpoint to get authorized workspaces is:
	// GET https://api.clickup.com/api/v2/team
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/team", nil)
	if err != nil {
		log.Errorf("Failed to create request: %v", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Failed to execute request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Process response
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("ClickUp API returned non-OK status: %d", resp.StatusCode)
		log.Error(err)
		return nil, err
	}

	// Parse response body
	var response struct {
		Teams []Workspace `json:"teams"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Errorf("Failed to decode response: %v", err)
		return nil, err
	}

	return response.Teams, nil
}

// GetSpaces retrieves the spaces in a workspace
func (s *ClientImpl) GetSpaces(ctx context.Context, workspaceId string) ([]Space, error) {
	client, err := s.prepareClickUpClient(ctx)
	if err != nil {
		log.Errorf("Failed to prepare ClickUp client: %v", err)
		return nil, err
	}

	// According to ClickUp API docs, the endpoint to get spaces is:
	// GET https://api.clickup.com/api/v2/team/{team_id}/space
	url := fmt.Sprintf("%s/team/%s/space", baseURL, workspaceId)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Errorf("Failed to create request: %v", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Failed to execute request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Process response
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("ClickUp API returned non-OK status: %d", resp.StatusCode)
		log.Error(err)
		return nil, err
	}

	// Parse response body
	var response struct {
		Spaces []Space `json:"spaces"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Errorf("Failed to decode response: %v", err)
		return nil, err
	}

	return response.Spaces, nil
}

// GetFolders retrieves the folders in a space
func (s *ClientImpl) GetFolders(ctx context.Context, spaceId string) ([]Folder, error) {
	client, err := s.prepareClickUpClient(ctx)
	if err != nil {
		log.Errorf("Failed to prepare ClickUp client: %v", err)
		return nil, err
	}

	// According to ClickUp API docs, the endpoint to get folders is:
	// GET https://api.clickup.com/api/v2/space/{space_id}/folder
	url := fmt.Sprintf("%s/space/%s/folder", baseURL, spaceId)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Errorf("Failed to create request: %v", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Failed to execute request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Process response
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("ClickUp API returned non-OK status: %d", resp.StatusCode)
		log.Error(err)
		return nil, err
	}

	// Parse response body
	var response struct {
		Folders []Folder `json:"folders"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Errorf("Failed to decode response: %v", err)
		return nil, err
	}

	return response.Folders, nil
}

// GetFilteredTeamTasks retrieves tasks for a team with optional filtering
func (s *ClientImpl) GetFilteredTeamTasks(ctx context.Context, workspaceId string, spaceId string, folderId string, page int, tagName string,
	withPrioritySetOnly bool) ([]Task, error) {

	client, err := s.prepareClickUpClient(ctx)
	if err != nil {
		log.Errorf("Failed to prepare ClickUp client: %v", err)
		return nil, err
	}

	// According to ClickUp API docs, the endpoint to get team tasks is:
	// GET https://api.clickup.com/api/v2/team/{team_id}/task
	url := fmt.Sprintf("%s/team/%s/task", baseURL, workspaceId)

	// Add query parameters for filtering
	queryParams := make(map[string]string)

	// Add space_ids query param if spaceId is provided
	if spaceId != "" {
		queryParams["space_ids[]"] = spaceId
	}

	// Add project_ids query param if folderId is provided
	if folderId != "" {
		queryParams["project_ids[]"] = folderId
	}

	// Add page query param
	queryParams["page"] = fmt.Sprintf("%d", page)

	// Add tag query param if tagName is provided
	if tagName != "" {
		queryParams["tags[]"] = tagName
	}

	// Include all subtasks in the response
	queryParams["subtasks"] = "true"

	// Construct the URL with query parameters
	if len(queryParams) > 0 {
		url += "?"
		first := true
		for key, value := range queryParams {
			if !first {
				url += "&"
			}
			url += fmt.Sprintf("%s=%s", key, value)
			first = false
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Errorf("Failed to create request: %v", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Failed to execute request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Process response
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("ClickUp API returned non-OK status: %d", resp.StatusCode)
		log.Error(err)
		return nil, err
	}

	// Parse response body
	var response struct {
		Tasks []Task `json:"tasks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Errorf("Failed to decode response: %v", err)
		return nil, err
	}

	filteredTasks := make([]Task, 0)
	if withPrioritySetOnly {
		for _, task := range response.Tasks {
			if task.Priority.Priority != "" {
				filteredTasks = append(filteredTasks, task)
			}
		}
		return filteredTasks, nil
	}

	return response.Tasks, nil
}

// GetTags retrieves tags for a space
func (s *ClientImpl) GetTags(ctx context.Context, spaceId string) ([]Tag, error) {
	client, err := s.prepareClickUpClient(ctx)
	if err != nil {
		log.Errorf("Failed to prepare ClickUp client: %v", err)
		return nil, err
	}

	// According to ClickUp API docs, the endpoint to get tags is:
	// GET https://api.clickup.com/api/v2/space/{space_id}/tag
	url := fmt.Sprintf("%s/space/%s/tag", baseURL, spaceId)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Errorf("Failed to create request: %v", err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Failed to execute request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Process response
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("ClickUp API returned non-OK status: %d", resp.StatusCode)
		log.Error(err)
		return nil, err
	}

	// Parse response body
	var response struct {
		Tags []Tag `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Errorf("Failed to decode response: %v", err)
		return nil, err
	}

	return response.Tags, nil
}
