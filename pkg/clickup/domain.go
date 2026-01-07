package clickup

type Configuration struct {
	WorkspaceId           string
	SpaceId               string
	FolderId              string
	OnlyTasksWithPriority bool
	Mappings              []BudgetItemMapping
}

type BudgetItemMapping struct {
	ClickupSpaceId string
	ClickupTagName string
	BudgetItemId   int
	Position       int
}
