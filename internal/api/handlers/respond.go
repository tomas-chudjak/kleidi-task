package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ahoylog/kvik-tasks/internal/core"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, core.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, core.ErrProjectNotFound):
		status = http.StatusNotFound
	case errors.Is(err, core.ErrAlreadyExists):
		status = http.StatusConflict
	case errors.Is(err, core.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, core.ErrNoProject):
		status = http.StatusNotFound
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
