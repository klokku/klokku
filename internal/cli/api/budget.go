package api

import (
	"fmt"
)

func (c *Client) ListBudgetPlans() ([]BudgetPlanDTO, error) {
	var plans []BudgetPlanDTO
	if err := c.Get("/api/budgetplan", &plans); err != nil {
		return nil, err
	}
	return plans, nil
}

func (c *Client) GetBudgetPlan(planID int) (*BudgetPlanDTO, error) {
	var plan BudgetPlanDTO
	if err := c.Get(fmt.Sprintf("/api/budgetplan/%d", planID), &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func (c *Client) CreateBudgetPlan(name string) (*BudgetPlanDTO, error) {
	body, err := jsonBody(BudgetPlanDTO{Name: name})
	if err != nil {
		return nil, err
	}
	var plan BudgetPlanDTO
	if err := c.Post("/api/budgetplan", body, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func (c *Client) UpdateBudgetPlan(plan BudgetPlanDTO) (*BudgetPlanDTO, error) {
	body, err := jsonBody(plan)
	if err != nil {
		return nil, err
	}
	var updated BudgetPlanDTO
	if err := c.Put(fmt.Sprintf("/api/budgetplan/%d", plan.ID), body, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteBudgetPlan(planID int) error {
	return c.Delete(fmt.Sprintf("/api/budgetplan/%d", planID))
}

func (c *Client) CreateBudgetItem(planID int, item BudgetItemDTO) (*BudgetItemDTO, error) {
	body, err := jsonBody(item)
	if err != nil {
		return nil, err
	}
	var created BudgetItemDTO
	if err := c.Post(fmt.Sprintf("/api/budgetplan/%d/item", planID), body, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateBudgetItem(planID int, item BudgetItemDTO) (*BudgetItemDTO, error) {
	body, err := jsonBody(item)
	if err != nil {
		return nil, err
	}
	var updated BudgetItemDTO
	if err := c.Put(fmt.Sprintf("/api/budgetplan/%d/item/%d", planID, item.ID), body, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteBudgetItem(planID, itemID int) error {
	return c.Delete(fmt.Sprintf("/api/budgetplan/%d/item/%d", planID, itemID))
}

func (c *Client) ReorderBudgetItem(planID, itemID, precedingID int) error {
	body, err := jsonBody(SetItemPositionRequest{ID: itemID, PrecedingID: precedingID})
	if err != nil {
		return err
	}
	req, err := c.newRequest("PUT", fmt.Sprintf("/api/budgetplan/%d/item/%d/position", planID, itemID), body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}
