// internal/handler/instructor.go
package handler

import (
	"api-server/internal/model"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type InstructorHandler struct {
	db *sql.DB
}

func NewInstructorHandler(db *sql.DB) *InstructorHandler {
	return &InstructorHandler{db: db}
}

func (h *InstructorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Instructor handler hit:", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")

	// Allow requests without authentication
	if r.Method == http.MethodGet {
		h.GetInstructorByID(w, r)
		return
	}

	// For all requests, require authentication
	// Get Basic Auth credentials
	username, password, hasAuth := r.BasicAuth()
	if !hasAuth {
		w.Header().Set("WWW-Authenticate", `Basic realm="Instructor Authentication Required"`)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
		return
	}

	// Authenticate user
	user, err := model.AuthenticateUser(h.db, username, password)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="Invalid Credentials"`)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid username or password"})
		return
	}

	// Check if user has instructor or admin role
	if user.Role != "admin" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Insufficient permissions"})
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.createInstructor(w, r, user)
	case http.MethodDelete:
		h.DeleteInstructorByID(w, r)
	case http.MethodPatch:
		h.PatchInstructor(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func (h *InstructorHandler) createInstructor(w http.ResponseWriter, r *http.Request, user *model.User) {
	var req model.CreateInstructorRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if err := req.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Use the authenticated user's ID as the user_id for the instructor
	instructor, err := model.CreateInstructor(h.db, req, user.ID)
	if err != nil {
		// Check for unique constraint violations
		if err.Error() == "pq: duplicate key value violates unique constraint \"instructors_email_key\"" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Email already exists"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create instructor"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(instructor)
}

func (h *InstructorHandler) GetInstructorByID(w http.ResponseWriter, r *http.Request) {
	// Get the instructor ID from query parameter
	instructorID := r.URL.Query().Get("id")

	// If no ID is provided, return an error
	if instructorID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Instructor ID is required"})
		return
	}

	// Process the provided ID
	id, err := uuid.Parse(instructorID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid instructor ID format"})
		return
	}

	instructor, err := model.GetInstructorByID(h.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Instructor not found"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve instructor"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(instructor)
}

func (h *InstructorHandler) DeleteInstructorByID(w http.ResponseWriter, r *http.Request) {
	// Get the instructor ID from query parameter
	instructorID := r.URL.Query().Get("id")

	// If no ID is provided, return an error
	if instructorID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Instructor ID is required"})
		return
	}

	// Parse the ID
	id, err := uuid.Parse(instructorID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid instructor ID format"})
		return
	}

	// Delete the instructor
	err = model.DeleteInstructorByID(h.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Instructor not found"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete instructor"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Instructor deleted successfully"})
}

func (h *InstructorHandler) PatchInstructor(w http.ResponseWriter, r *http.Request) {
	// Get the instructor ID from query parameter
	instructorID := r.URL.Query().Get("id")

	// If no ID is provided, return an error
	if instructorID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Instructor ID is required"})
		return
	}

	// Parse the ID
	id, err := uuid.Parse(instructorID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid instructor ID format"})
		return
	}

	// Parse the update request
	var updateReq model.UpdateInstructorRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Update the instructor
	updatedInstructor, err := model.UpdateInstructor(h.db, id, updateReq)
	if err != nil {
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "unique constraint") && strings.Contains(err.Error(), "email") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Email already exists"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update instructor"})
		return
	}

	// Return the updated instructor
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedInstructor)
}
