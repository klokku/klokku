package clickup

type Configuration struct {
	WorkspaceId           int
	SpaceId               int
	FolderId              int
	OnlyTasksWithPriority bool
	Mappings              []BudgetMapping
}

type BudgetMapping struct {
	Id             int
	ClickupSpaceId int
	ClickupTagName string
	BudgetItemId   int
	Position       int
}
