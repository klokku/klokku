package webhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type WebhookDTO struct {
	Id         int             `json:"id"`
	Type       WebhookType     `json:"type"`
	Token      string          `json:"token"`
	WebhookUrl string          `json:"webhookUrl"`
	Data       json.RawMessage `json:"data" swaggertype:"object"`
}
type Handler struct {
	appHost string
	service Service
}

func NewHandler(appHost string, service Service) *Handler {
	return &Handler{
		appHost: appHost,
		service: service,
	}
}

// HandleWebhook godoc
// @Summary Handle incoming webhook
// @Description Execute a webhook action using the webhook token (no authentication required)
// @Tags Webhook
// @Accept json
// @Produce json
// @Param token path string true "Webhook Token"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Invalid webhook token"
// @Router /webhook/{token} [post]
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Debug("Webhook request received")

	vars := mux.Vars(r)
	token := vars["token"]

	if token == "" {
		http.Error(w, "Missing webhook token", http.StatusBadRequest)
		return
	}

	// Execute webhook
	err := h.service.Execute(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrWebhookNotFound) {
			http.Error(w, "Invalid webhook token", http.StatusNotFound)
			return
		}
		log.Errorf("Failed to execute webhook: %v", err)
		http.Error(w, "Failed to execute webhook", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   "Webhook executed successfully",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// CreateWebhook godoc
// @Summary Create a new webhook
// @Description Create a new webhook for a specific action
// @Tags Webhook
// @Accept json
// @Produce json
// @Param webhook body object{type=string,data=object} true "Webhook creation request"
// @Success 201 {object} WebhookDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/webhook [post]
// @Security XUserId
func (h *Handler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request struct {
		Type WebhookType           `json:"type"`
		Data StartCurrentEventData `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate webhook type
	if request.Type != TypeStartCurrentEvent {
		http.Error(w, "Invalid webhook type", http.StatusBadRequest)
		return
	}

	webhook, err := h.service.Create(r.Context(), request.Type, request.Data)
	if err != nil {
		log.Errorf("Failed to create webhook: %v", err)
		http.Error(w, "Failed to create webhook", http.StatusInternalServerError)
		return
	}

	response := h.webhookToDTO(webhook)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ListWebhooks godoc
// @Summary List user's webhooks
// @Description Get all webhooks for the current user filtered by type
// @Tags Webhook
// @Produce json
// @Param type query string true "Webhook type"
// @Success 200 {array} WebhookDTO
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Router /api/webhook [get]
// @Security XUserId
func (h *Handler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	webhookType := WebhookType(r.URL.Query().Get("type"))
	if webhookType == "" {
		http.Error(w, "Missing webhook type parameter", http.StatusBadRequest)
		return
	}

	webhooks, err := h.service.GetByUserIdAndType(r.Context(), webhookType)
	if err != nil {
		log.Errorf("Failed to list webhooks: %v", err)
		http.Error(w, "Failed to list webhooks", http.StatusInternalServerError)
		return
	}

	dtos := make([]WebhookDTO, len(webhooks))
	for i := range webhooks {
		dtos[i] = h.webhookToDTO(webhooks[i])
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(dtos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteWebhook godoc
// @Summary Delete a webhook
// @Description Delete a webhook by ID
// @Tags Webhook
// @Param id path int true "Webhook ID"
// @Success 204 "No Content"
// @Failure 400 {string} string "Bad Request"
// @Failure 403 {string} string "User not found"
// @Failure 404 {string} string "Webhook not found"
// @Router /api/webhook/{id} [delete]
// @Security XUserId
func (h *Handler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookIdStr := vars["id"]
	webhookId, err := strconv.Atoi(webhookIdStr)
	if err != nil {
		http.Error(w, "Invalid webhook ID", http.StatusBadRequest)
		return
	}

	err = h.service.Delete(r.Context(), webhookId)
	if err != nil {
		if errors.Is(err, ErrWebhookNotFound) {
			http.Error(w, "Webhook not found", http.StatusNotFound)
			return
		}
		log.Errorf("Failed to delete webhook: %v", err)
		http.Error(w, "Failed to delete webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) webhookToDTO(webhook Webhook) WebhookDTO {
	return WebhookDTO{
		Id:         webhook.Id,
		Type:       webhook.Type,
		Token:      webhook.Token,
		WebhookUrl: fmt.Sprintf("%s/api/webhook/%s", h.appHost, webhook.Token),
		Data:       webhook.Data,
	}
}
