package weekly_plan

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type RepositoryStub struct {
	mu             sync.RWMutex
	items          map[int]WeeklyPlanItem // id -> item
	userIds        map[int]int            // id -> userId
	nextId         int
	weeklyPlans    map[string]WeeklyPlan // "userId:weekNumber" -> plan
	nextPlanId     int
	inTransaction  bool
	transactionErr error
}

func NewRepositoryStub() *RepositoryStub {
	return &RepositoryStub{
		items:       make(map[int]WeeklyPlanItem),
		userIds:     make(map[int]int),
		nextId:      1,
		weeklyPlans: make(map[string]WeeklyPlan),
		nextPlanId:  1,
	}
}

func weeklyPlanKey(userId int, weekNumber WeekNumber) string {
	return fmt.Sprintf("%d:%s", userId, weekNumber.String())
}

func (r *RepositoryStub) WithTransaction(ctx context.Context, fn func(repo Repository) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create a copy of the current state for rollback
	originalItems := make(map[int]WeeklyPlanItem, len(r.items))
	for k, v := range r.items {
		originalItems[k] = v
	}
	originalUserIds := make(map[int]int)
	for k, v := range r.userIds {
		originalUserIds[k] = v
	}
	originalNextId := r.nextId
	originalWeeklyPlans := make(map[string]WeeklyPlan, len(r.weeklyPlans))
	for k, v := range r.weeklyPlans {
		originalWeeklyPlans[k] = v
	}
	originalNextPlanId := r.nextPlanId

	// Mark as in transaction
	r.inTransaction = true
	r.transactionErr = nil
	r.mu.Unlock()

	// Execute the function
	err := fn(r)

	r.mu.Lock()
	r.inTransaction = false

	// Rollback on error
	if err != nil || r.transactionErr != nil {
		r.items = originalItems
		r.userIds = originalUserIds
		r.nextId = originalNextId
		r.weeklyPlans = originalWeeklyPlans
		r.nextPlanId = originalNextPlanId
		if err != nil {
			return err
		}
		return r.transactionErr
	}

	return nil
}

func (r *RepositoryStub) GetItemsForWeek(ctx context.Context, userId int, weekNumber WeekNumber) ([]WeeklyPlanItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []WeeklyPlanItem
	for id, item := range r.items {
		if r.userIds[id] == userId && item.WeekNumber == weekNumber {
			result = append(result, item)
		}
	}

	// Sort by position (simple bubble sort for small slices)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Position > result[j].Position {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

func (r *RepositoryStub) GetItem(ctx context.Context, userId int, id int) (WeeklyPlanItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, exists := r.items[id]
	if !exists || r.userIds[id] != userId {
		return WeeklyPlanItem{}, ErrWeeklyPlanItemNotFound
	}

	return item, nil
}

func (r *RepositoryStub) UpdateAllItemsByBudgetItemId(
	ctx context.Context,
	userId int,
	budgetItemId int,
	name string,
	icon string,
	color string,
) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for id, item := range r.items {
		if r.userIds[id] == userId && item.BudgetItemId == budgetItemId {
			item.Name = name
			item.Icon = icon
			item.Color = color
			r.items[id] = item
			count++
		}
	}

	return count, nil
}

func (r *RepositoryStub) UpdateItem(
	ctx context.Context,
	userId int,
	id int,
	weeklyDuration time.Duration,
	notes string,
) (WeeklyPlanItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, exists := r.items[id]
	if !exists || r.userIds[id] != userId {
		return WeeklyPlanItem{}, ErrWeeklyPlanItemNotFound
	}

	item.WeeklyDuration = weeklyDuration
	item.Notes = notes
	r.items[id] = item

	return item, nil
}

func (r *RepositoryStub) createItems(ctx context.Context, userId int, items []WeeklyPlanItem) ([]WeeklyPlanItem, error) {
	if len(items) == 0 {
		return nil, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	var created []WeeklyPlanItem
	for _, item := range items {
		item.Id = r.nextId
		r.items[r.nextId] = item
		r.userIds[r.nextId] = userId
		r.nextId++
		created = append(created, item)
	}

	return created, nil
}

func (r *RepositoryStub) DeleteWeekItems(ctx context.Context, userId int, weekNumber WeekNumber) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for id, item := range r.items {
		if r.userIds[id] == userId && item.WeekNumber == weekNumber {
			delete(r.items, id)
			delete(r.userIds, id)
			count++
		}
	}

	return count, nil
}

// Helper method to set transaction error (for testing transaction rollback)
func (r *RepositoryStub) SetTransactionError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transactionErr = err
}

// Helper method to get all items (useful for test assertions)
func (r *RepositoryStub) GetAllItems() []WeeklyPlanItem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]WeeklyPlanItem, 0, len(r.items))
	for _, item := range r.items {
		result = append(result, item)
	}
	return result
}

func (r *RepositoryStub) GetWeeklyPlan(ctx context.Context, userId int, weekNumber WeekNumber) (*WeeklyPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := weeklyPlanKey(userId, weekNumber)
	if wp, ok := r.weeklyPlans[key]; ok {
		return &wp, nil
	}
	return nil, nil
}

func (r *RepositoryStub) CreateWeeklyPlan(ctx context.Context, userId int, budgetPlanId int, weekNumber WeekNumber) (WeeklyPlan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := weeklyPlanKey(userId, weekNumber)
	wp := WeeklyPlan{
		Id:           r.nextPlanId,
		BudgetPlanId: budgetPlanId,
		WeekNumber:   weekNumber,
		IsOffWeek:    false,
	}
	r.weeklyPlans[key] = wp
	r.nextPlanId++
	return wp, nil
}

func (r *RepositoryStub) SetOffWeek(ctx context.Context, userId int, budgetPlanId int, weekNumber WeekNumber, isOffWeek bool) (WeeklyPlan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := weeklyPlanKey(userId, weekNumber)
	wp, ok := r.weeklyPlans[key]
	if !ok {
		wp = WeeklyPlan{
			Id:           r.nextPlanId,
			BudgetPlanId: budgetPlanId,
			WeekNumber:   weekNumber,
		}
		r.nextPlanId++
	}
	wp.IsOffWeek = isOffWeek
	r.weeklyPlans[key] = wp
	return wp, nil
}

func (r *RepositoryStub) DeleteWeeklyPlan(ctx context.Context, userId int, weekNumber WeekNumber) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := weeklyPlanKey(userId, weekNumber)
	delete(r.weeklyPlans, key)
	return nil
}

// Helper method to reset the stub (useful between tests)
func (r *RepositoryStub) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = make(map[int]WeeklyPlanItem)
	r.userIds = make(map[int]int)
	r.nextId = 1
	r.weeklyPlans = make(map[string]WeeklyPlan)
	r.nextPlanId = 1
	r.inTransaction = false
	r.transactionErr = nil
}
