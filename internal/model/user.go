// internal/model/user.go
package model

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Username       string    `json:"username"`
	Password       string    `json:"-"` // Don't expose password in JSON responses
	Role           string    `json:"role"`
	Email          string    `json:"email"`
	AccountCreated time.Time `json:"account_created"`
	AccountUpdated time.Time `json:"account_updated"`
}

type CreateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	Email     string `json:"email"`
}

type UpdateUserRequest struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
}

func (r *CreateUserRequest) Validate() error {
	if r.FirstName == "" {
		return errors.New("first name is required")
	}
	if r.Username == "" {
		return errors.New("username is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	if r.Role != "student" && r.Role != "admin" && r.Role != "instructor" {
		return errors.New("role must be student, admin, or instructor")
	}
	if r.Email == "" {
		return errors.New("email is required")
	}

	// Email format validation using regex
	emailRegex := regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)
	if !emailRegex.MatchString(r.Email) {
		return errors.New("invalid email format")
	}

	return nil
}

func CreateUser(db *sql.DB, req CreateUserRequest) (*User, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var user User
	query := `
        INSERT INTO webapp.users (first_name, last_name, username, password, role, email)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, first_name, last_name, username, role, email, account_created, account_updated
    `

	err = db.QueryRow(
		query,
		req.FirstName,
		req.LastName,
		req.Username,
		string(hashedPassword),
		req.Role,
		req.Email,
	).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Role,
		&user.Email,
		&user.AccountCreated,
		&user.AccountUpdated,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func AuthenticateUser(db *sql.DB, username, password string) (*User, error) {
	var user User
	var hashedPassword string

	query := `
        SELECT id, first_name, last_name, username, password, role, email, account_created, account_updated 
        FROM webapp.users 
        WHERE username = $1
    `

	err := db.QueryRow(query, username).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&hashedPassword,
		&user.Role,
		&user.Email,
		&user.AccountCreated,
		&user.AccountUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	return &user, nil
}

func UpdateUser(db *sql.DB, userID uuid.UUID, req UpdateUserRequest) (*User, error) {
	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Build the update query dynamically based on which fields are provided
	query := "UPDATE webapp.users SET"
	args := []interface{}{userID}
	argIndex := 2 // Start at 2 because userID is $1

	// Track if we need to add fields
	var updates []string

	if req.FirstName != "" {
		updates = append(updates, fmt.Sprintf(" first_name = $%d", argIndex))
		args = append(args, req.FirstName)
		argIndex++
	}

	if req.LastName != "" {
		updates = append(updates, fmt.Sprintf(" last_name = $%d", argIndex))
		args = append(args, req.LastName)
		argIndex++
	}

	if req.Username != "" {
		updates = append(updates, fmt.Sprintf(" username = $%d", argIndex))
		args = append(args, req.Username)
		argIndex++
	}

	if req.Password != "" {
		// Hash the new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		updates = append(updates, fmt.Sprintf(" password = $%d", argIndex))
		args = append(args, string(hashedPassword))
		argIndex++
	}

	// Add account_updated timestamp
	updates = append(updates, " account_updated = CURRENT_TIMESTAMP")

	// If no fields to update, return the current user
	if len(updates) == 1 { // Only timestamp update
		return GetUserByID(db, userID)
	}

	// Complete the query
	query += strings.Join(updates, ",")
	query += " WHERE id = $1 RETURNING id, first_name, last_name, username, role, email, account_created, account_updated"

	// Execute the update
	var user User
	err = tx.QueryRow(query, args...).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Role,
		&user.Email,
		&user.AccountCreated,
		&user.AccountUpdated,
	)

	if err != nil {
		return nil, err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &user, nil
}

// helper function to get a user by ID
func GetUserByID(db *sql.DB, userID uuid.UUID) (*User, error) {
	var user User

	query := `
        SELECT id, first_name, last_name, username, role, email, account_created, account_updated 
        FROM webapp.users 
        WHERE id = $1
    `

	err := db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Role,
		&user.Email,
		&user.AccountCreated,
		&user.AccountUpdated,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
