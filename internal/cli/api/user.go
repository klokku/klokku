package api

// GetCurrentUser returns the currently authenticated user.
func (c *Client) GetCurrentUser() (*UserDTO, error) {
	var user UserDTO
	if err := c.Get("/api/user/current", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers returns all registered users.
func (c *Client) ListUsers() ([]UserDTO, error) {
	var users []UserDTO
	if err := c.Get("/api/user", &users); err != nil {
		return nil, err
	}
	return users, nil
}
