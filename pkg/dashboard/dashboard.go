// Package dashboard stores profile-specific dashboard preferences independently of other user settings.
package dashboard

type Configuration struct {
	Cards []Card `json:"cards"`
}

type Card struct {
	Id      string `json:"id"`
	Visible bool   `json:"visible"`
}
