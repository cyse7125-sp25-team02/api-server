// internal/handler/health.go
package handler

import (
	"api-server/internal/model"
	"database/sql"
	"io"
	"net/http"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set no-cache header
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", "application/json")

	// Check if method is GET
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Check for any query parameters
	if len(r.URL.Query()) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check for any path parameters
	if r.URL.Path != "/healthz" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check for payload in request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(body) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Insert health check record
	err = model.InsertHealthCheck(h.db)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}
