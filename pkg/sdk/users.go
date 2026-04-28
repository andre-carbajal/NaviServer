package sdk

import "fmt"

func (c *Client) ListUsers() ([]User, error) {
	var users []User
	err := c.get("/users", &users)
	return users, err
}

func (c *Client) CreateUser(username, password string) (*User, error) {
	payload := map[string]string{
		"username": username,
		"password": password,
	}

	var user User
	err := c.post("/users", payload, &user)
	return &user, err
}

func (c *Client) DeleteUser(id string) error {
	return c.delete(fmt.Sprintf("/users/%s", id))
}

func (c *Client) UpdatePassword(id, password string) error {
	payload := map[string]string{"password": password}
	return c.put(fmt.Sprintf("/users/%s/password", id), payload)
}

func (c *Client) GetPermissions(userID string) ([]Permission, error) {
	var permissions []Permission
	err := c.get(fmt.Sprintf("/users/%s/permissions", userID), &permissions)
	return permissions, err
}

func (c *Client) SetPermissions(permissions []Permission) error {
	return c.put("/users/permissions", permissions)
}
