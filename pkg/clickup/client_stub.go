package clickup

import (
	"context"
	"errors"
	"sync"
)

type ClientStub struct {
	mu                         sync.RWMutex
	workspaces                 []Workspace
	spaces                     map[string][]Space  // workspaceId -> spaces
	folders                    map[string][]Folder // spaceId -> folders
	tags                       map[string][]Tag    // spaceId -> tags
	tasks                      map[taskKey][]Task
	getAuthorizedWorkspacesErr error
	getSpacesErr               error
	getFoldersErr              error
	getTagsErr                 error
	getFilteredTeamTasksErr    error
}

type taskKey struct {
	workspaceId         string
	spaceId             string
	folderId            string
	page                int
	tagName             string
	withPrioritySetOnly bool
}

func NewClientStub() *ClientStub {
	return &ClientStub{
		spaces:  make(map[string][]Space),
		folders: make(map[string][]Folder),
		tags:    make(map[string][]Tag),
		tasks:   make(map[taskKey][]Task),
	}
}

func (c *ClientStub) GetAuthorizedWorkspaces(ctx context.Context) ([]Workspace, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.getAuthorizedWorkspacesErr != nil {
		return nil, c.getAuthorizedWorkspacesErr
	}

	result := make([]Workspace, len(c.workspaces))
	copy(result, c.workspaces)
	return result, nil
}

func (c *ClientStub) GetSpaces(ctx context.Context, workspaceId string) ([]Space, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.getSpacesErr != nil {
		return nil, c.getSpacesErr
	}

	spaces, exists := c.spaces[workspaceId]
	if !exists {
		return []Space{}, nil
	}

	result := make([]Space, len(spaces))
	copy(result, spaces)
	return result, nil
}

func (c *ClientStub) GetFolders(ctx context.Context, spaceId string) ([]Folder, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.getFoldersErr != nil {
		return nil, c.getFoldersErr
	}

	folders, exists := c.folders[spaceId]
	if !exists {
		return []Folder{}, nil
	}

	result := make([]Folder, len(folders))
	copy(result, folders)
	return result, nil
}

func (c *ClientStub) GetFilteredTeamTasks(
	ctx context.Context,
	workspaceId string,
	spaceId string,
	folderId string,
	page int,
	tagName string,
	withPrioritySetOnly bool,
) ([]Task, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.getFilteredTeamTasksErr != nil {
		return nil, c.getFilteredTeamTasksErr
	}

	key := taskKey{
		workspaceId:         workspaceId,
		spaceId:             spaceId,
		folderId:            folderId,
		page:                page,
		tagName:             tagName,
		withPrioritySetOnly: withPrioritySetOnly,
	}

	tasks, exists := c.tasks[key]
	if !exists {
		return []Task{}, nil
	}

	result := make([]Task, len(tasks))
	copy(result, tasks)
	return result, nil
}

func (c *ClientStub) GetTags(ctx context.Context, spaceId string) ([]Tag, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.getTagsErr != nil {
		return nil, c.getTagsErr
	}

	tags, exists := c.tags[spaceId]
	if !exists {
		return []Tag{}, nil
	}

	result := make([]Tag, len(tags))
	copy(result, tags)
	return result, nil
}

// Helper methods for test setup

func (c *ClientStub) SetWorkspaces(workspaces []Workspace) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.workspaces = make([]Workspace, len(workspaces))
	copy(c.workspaces, workspaces)
}

func (c *ClientStub) SetSpaces(workspaceId string, spaces []Space) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.spaces[workspaceId] = make([]Space, len(spaces))
	copy(c.spaces[workspaceId], spaces)
}

func (c *ClientStub) SetFolders(spaceId string, folders []Folder) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.folders[spaceId] = make([]Folder, len(folders))
	copy(c.folders[spaceId], folders)
}

func (c *ClientStub) SetTags(spaceId string, tags []Tag) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tags[spaceId] = make([]Tag, len(tags))
	copy(c.tags[spaceId], tags)
}

func (c *ClientStub) SetTasks(
	workspaceId string,
	spaceId string,
	folderId string,
	page int,
	tagName string,
	withPrioritySetOnly bool,
	tasks []Task,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := taskKey{
		workspaceId:         workspaceId,
		spaceId:             spaceId,
		folderId:            folderId,
		page:                page,
		tagName:             tagName,
		withPrioritySetOnly: withPrioritySetOnly,
	}

	c.tasks[key] = make([]Task, len(tasks))
	copy(c.tasks[key], tasks)
}

// Error setters for testing error scenarios

func (c *ClientStub) SetGetAuthorizedWorkspacesError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getAuthorizedWorkspacesErr = err
}

func (c *ClientStub) SetGetSpacesError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getSpacesErr = err
}

func (c *ClientStub) SetGetFoldersError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getFoldersErr = err
}

func (c *ClientStub) SetGetTagsError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getTagsErr = err
}

func (c *ClientStub) SetGetFilteredTeamTasksError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getFilteredTeamTasksErr = err
}

// Reset clears all data
func (c *ClientStub) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.workspaces = nil
	c.spaces = make(map[string][]Space)
	c.folders = make(map[string][]Folder)
	c.tags = make(map[string][]Tag)
	c.tasks = make(map[taskKey][]Task)
	c.getAuthorizedWorkspacesErr = nil
	c.getSpacesErr = nil
	c.getFoldersErr = nil
	c.getTagsErr = nil
	c.getFilteredTeamTasksErr = nil
}

var ErrClientTestError = errors.New("client test error")
