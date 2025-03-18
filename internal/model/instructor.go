// internal/model/instructor.go
package model

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Instructor struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	DateAdded   time.Time `json:"date_added"`
	DateUpdated time.Time `json:"date_updated"`
}

type CreateInstructorRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateInstructorRequest struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

func (r *CreateInstructorRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
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

func CreateInstructor(db *sql.DB, req CreateInstructorRequest, userID uuid.UUID) (*Instructor, error) {
	var instructor Instructor
	query := `
	INSERT INTO webapp.instructors (user_id, name, email)
	VALUES ($1, $2, $3)
	RETURNING id, user_id, name, email, date_added, date_updated
	`

	err := db.QueryRow(
		query,
		userID,
		req.Name,
		req.Email,
	).Scan(
		&instructor.ID,
		&instructor.UserID,
		&instructor.Name,
		&instructor.Email,
		&instructor.DateAdded,
		&instructor.DateUpdated,
	)

	if err != nil {
		return nil, err
	}

	return &instructor, nil
}

func GetInstructorByID(db *sql.DB, instructorID uuid.UUID) (*Instructor, error) {
	var instructor Instructor

	query := `
	SELECT id, user_id, name, email, date_added, date_updated
	FROM webapp.instructors
	WHERE id = $1
	`

	err := db.QueryRow(query, instructorID).Scan(
		&instructor.ID,
		&instructor.UserID,
		&instructor.Name,
		&instructor.Email,
		&instructor.DateAdded,
		&instructor.DateUpdated,
	)

	if err != nil {
		return nil, err
	}

	return &instructor, nil
}

func DeleteInstructorByID(db *sql.DB, instructorID uuid.UUID) error {
	query := `
	DELETE FROM webapp.instructors
	WHERE id = $1
	`

	result, err := db.Exec(query, instructorID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func UpdateInstructor(db *sql.DB, instructorID uuid.UUID, req UpdateInstructorRequest) (*Instructor, error) {
	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Build the update query dynamically based on which fields are provided
	query := "UPDATE webapp.instructors SET"
	args := []interface{}{instructorID}
	argIndex := 2 // Start at 2 because instructorID is $1

	// Track if we need to add fields
	var updates []string

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf(" name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.Email != nil {
		updates = append(updates, fmt.Sprintf(" email = $%d", argIndex))
		args = append(args, *req.Email)
		argIndex++
	}

	// Add date_updated timestamp
	updates = append(updates, " date_updated = CURRENT_TIMESTAMP")

	// If no fields to update, return the current instructor
	if len(updates) == 1 { // Only timestamp update
		return GetInstructorByID(db, instructorID)
	}

	// Complete the query
	query += strings.Join(updates, ",")
	query += " WHERE id = $1 RETURNING id, user_id, name, email, date_added, date_updated"

	// Execute the update
	var instructor Instructor
	err = tx.QueryRow(query, args...).Scan(
		&instructor.ID,
		&instructor.UserID,
		&instructor.Name,
		&instructor.Email,
		&instructor.DateAdded,
		&instructor.DateUpdated,
	)

	if err != nil {
		return nil, err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &instructor, nil
}
