package dashboard

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetConfiguration godoc
// @Summary Get dashboard configuration (null means application defaults)
// @Tags Dashboard
// @Produce json
// @Success 200 {object} Configuration
// @Failure 404 {string} string "User not found"
// @Failure 500 {string} string "Failed to read configuration"
// @Router /api/user/current/dashboard [get]
// @Security XUserId
func (h *Handler) GetConfiguration(w http.ResponseWriter, r *http.Request) {
	configuration, err := h.service.FindConfiguration(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(configuration); err != nil {
		http.Error(w, "Failed to encode dashboard configuration", http.StatusInternalServerError)
	}
}

// SaveConfiguration godoc
// @Summary Save ordered dashboard card visibility
// @Tags Dashboard
// @Accept json
// @Produce json
// @Param configuration body Configuration true "Dashboard configuration"
// @Success 200 {object} Configuration
// @Failure 400 {string} string "Invalid configuration"
// @Failure 404 {string} string "User not found"
// @Failure 500 {string} string "Failed to save configuration"
// @Router /api/user/current/dashboard [put]
// @Security XUserId
func (h *Handler) SaveConfiguration(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Cards []struct {
			Id      string `json:"id"`
			Visible *bool  `json:"visible"`
		} `json:"cards"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.Cards == nil {
		http.Error(w, "Invalid dashboard configuration", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		http.Error(w, "Invalid dashboard configuration", http.StatusBadRequest)
		return
	}
	configuration := Configuration{Cards: make([]Card, 0, len(payload.Cards))}
	for _, card := range payload.Cards {
		if card.Visible == nil {
			http.Error(w, "Card visibility is required", http.StatusBadRequest)
			return
		}
		configuration.Cards = append(configuration.Cards, Card{Id: card.Id, Visible: *card.Visible})
	}
	if err := h.service.SaveConfiguration(r.Context(), &configuration); err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(configuration); err != nil {
		http.Error(w, "Failed to encode dashboard configuration", http.StatusInternalServerError)
	}
}

// ResetConfiguration godoc
// @Summary Restore application dashboard defaults
// @Tags Dashboard
// @Success 204
// @Failure 404 {string} string "User not found"
// @Failure 500 {string} string "Failed to reset configuration"
// @Router /api/user/current/dashboard [delete]
// @Security XUserId
func (h *Handler) ResetConfiguration(w http.ResponseWriter, r *http.Request) {
	if err := h.service.SaveConfiguration(r.Context(), nil); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrConfigurationInvalid) {
		http.Error(w, "Invalid dashboard configuration", http.StatusBadRequest)
		return
	}
	if errors.Is(err, user.ErrUserNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	log.WithError(err).Error("dashboard configuration request failed")
	http.Error(w, "Dashboard configuration request failed", http.StatusInternalServerError)
}
