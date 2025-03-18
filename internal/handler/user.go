// internal/handler/user.go
package handler

import (
	"api-server/internal/model"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type UserHandler struct {
	db *sql.DB
}

func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("User handler hit:", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		h.GetUser(w, r)
	case http.MethodPost:
		h.createUser(w, r)
	case http.MethodPut:
		h.UpdateUser(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
	}
}

func (h *UserHandler) createUser(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest

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

	user, err := model.CreateUser(h.db, req)
	if err != nil {
		// Check for unique constraint violations
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_username_key\"" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Username already exists"})
			return
		}
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Email already exists"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get Basic Auth credentials
	username, password, hasAuth := r.BasicAuth()
	if !hasAuth {
		w.Header().Set("WWW-Authenticate", `Basic realm="User Authentication Required"`)
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

	// Return the authenticated user
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get Basic Auth credentials
	username, password, hasAuth := r.BasicAuth()
	if !hasAuth {
		w.Header().Set("WWW-Authenticate", `Basic realm="User Authentication Required"`)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
		return
	}

	// Authenticate user
	authenticatedUser, err := model.AuthenticateUser(h.db, username, password)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="Invalid Credentials"`)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid username or password"})
		return
	}

	// Parse the update request
	var updateReq model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Update the user
	updatedUser, err := model.UpdateUser(h.db, authenticatedUser.ID, updateReq)
	if err != nil {
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "unique constraint") && strings.Contains(err.Error(), "username") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Username already exists"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update user"})
		return
	}

	// Return the updated user
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedUser)
}
