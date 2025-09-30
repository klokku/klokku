package calendar_provider

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/klokku/klokku/internal/rest"
)

type MigrationStatusDTO struct {
	Status         string `json:"status"`
	MigratedEvents int    `json:"migratedEvents"`
}

type MigratorHandler struct {
	eventsMigrator *EventsMigratorImpl
}

func NewMigratorHandler(eventsMigrator *EventsMigratorImpl) *MigratorHandler {
	return &MigratorHandler{
		eventsMigrator: eventsMigrator,
	}
}

func (h *MigratorHandler) MigrateFromKlokkuToGoogle(w http.ResponseWriter, r *http.Request) {
	fromString := r.URL.Query().Get("from")
	toString := r.URL.Query().Get("to")
	from, err := time.Parse(time.RFC3339, fromString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid from (date) format",
			Details: "'from' must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	to, err := time.Parse(time.RFC3339, toString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid to (date) format",
			Details: "'to' must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	eventsMigrated, err := h.eventsMigrator.MigrateFromKlokkuToGoogle(r.Context(), from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	migrationStatus := MigrationStatusDTO{
		Status:         "COMPLETED",
		MigratedEvents: eventsMigrated,
	}
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(migrationStatus); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *MigratorHandler) MigrateFromGoogleToKlokku(w http.ResponseWriter, r *http.Request) {
	fromString := r.URL.Query().Get("from")
	toString := r.URL.Query().Get("to")
	from, err := time.Parse(time.RFC3339, fromString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid from (date) format",
			Details: "'from' must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	to, err := time.Parse(time.RFC3339, toString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := json.NewEncoder(w).Encode(rest.ErrorResponse{
			Error:   "Invalid to (date) format",
			Details: "'to' must be in RFC3339 format",
		})
		if encodeErr != nil {
			http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	eventsMigrated, err := h.eventsMigrator.MigrateFromGoogleToKlokku(r.Context(), from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	migrationStatus := MigrationStatusDTO{
		Status:         "COMPLETED",
		MigratedEvents: eventsMigrated,
	}
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(migrationStatus); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
