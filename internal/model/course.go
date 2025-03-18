// internal/model/course.go
package model

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Course struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	SemesterTerm string    `json:"semester_term"`
	CreditHours  int       `json:"credit_hours"`
	SubjectCode  string    `json:"subject_code"`
	CourseID     int       `json:"course_id"`
	SemesterYear int       `json:"semester_year"`
	DateCreated  time.Time `json:"date_created"`
	DateUpdated  time.Time `json:"date_updated"`
	UserID       uuid.UUID `json:"user_id"`
	InstructorID uuid.UUID `json:"instructor_id"`
}

type CreateCourseRequest struct {
	Name         string    `json:"name"`
	SemesterTerm string    `json:"semester_term"`
	CreditHours  int       `json:"credit_hours"`
	SubjectCode  string    `json:"subject_code"`
	CourseID     int       `json:"course_id"`
	SemesterYear int       `json:"semester_year"`
	InstructorID uuid.UUID `json:"instructor_id"`
}

// UpdateCourseRequest defines the optional fields for updating a course via PATCH.
type UpdateCourseRequest struct {
	Name         *string    `json:"name,omitempty"`
	SemesterTerm *string    `json:"semester_term,omitempty"`
	CreditHours  *int       `json:"credit_hours,omitempty"`
	SubjectCode  *string    `json:"subject_code,omitempty"`
	CourseID     *int       `json:"course_id,omitempty"`
	SemesterYear *int       `json:"semester_year,omitempty"`
	InstructorID *uuid.UUID `json:"instructor_id,omitempty"`
}

type Trace struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	InstructorID uuid.UUID `json:"instructor_id"`
	Status       string    `json:"status"`
	VectorID     *string   `json:"vector_id"`
	FileName     string    `json:"file_name"`
	BucketURL    string    `json:"bucket_url"`
	DateCreated  time.Time `json:"date_created"`
	DateUpdated  time.Time `json:"date_updated"`
}

func (r *CreateCourseRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.SemesterTerm != "Fall" && r.SemesterTerm != "Spring" && r.SemesterTerm != "Summer" {
		return errors.New("semester_term must be 'Fall', 'Spring', or 'Summer'")
	}
	if r.CreditHours <= 0 {
		return errors.New("credit_hours must be greater than 0")
	}
	if r.SubjectCode == "" {
		return errors.New("subject_code is required")
	}
	if r.CourseID < 1 || r.CourseID > 99999999 {
		return errors.New("course_id must be between 1 and 99999999")
	}
	if r.SemesterYear < 2000 {
		return errors.New("semester_year must be greater than or equal to 2000")
	}
	if r.InstructorID == uuid.Nil {
		return errors.New("instructor_id is required")
	}
	return nil
}

// Update Validate ensures the provided fields meet database constraints.
func (r *UpdateCourseRequest) Validate() error {
	if r.SemesterTerm != nil && *r.SemesterTerm != "Fall" && *r.SemesterTerm != "Spring" && *r.SemesterTerm != "Summer" {
		return errors.New("semester_term must be 'Fall', 'Spring', or 'Summer'")
	}
	if r.CreditHours != nil && *r.CreditHours <= 0 {
		return errors.New("credit_hours must be greater than 0")
	}
	if r.CourseID != nil && (*r.CourseID < 1 || *r.CourseID > 99999999) {
		return errors.New("course_id must be between 1 and 99999999")
	}
	if r.SemesterYear != nil && *r.SemesterYear < 2000 {
		return errors.New("semester_year must be greater than or equal to 2000")
	}
	return nil
}

func CreateCourse(db *sql.DB, req CreateCourseRequest, userID uuid.UUID) (*Course, error) {
	var course Course
	query := `
		INSERT INTO webapp.courses (name, semester_term, credit_hours, subject_code, course_id, semester_year, user_id, instructor_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, semester_term, credit_hours, subject_code, course_id, semester_year, date_created, date_updated, user_id, instructor_id
	`
	err := db.QueryRow(
		query,
		req.Name,
		req.SemesterTerm,
		req.CreditHours,
		req.SubjectCode,
		req.CourseID,
		req.SemesterYear,
		userID,
		req.InstructorID,
	).Scan(
		&course.ID,
		&course.Name,
		&course.SemesterTerm,
		&course.CreditHours,
		&course.SubjectCode,
		&course.CourseID,
		&course.SemesterYear,
		&course.DateCreated,
		&course.DateUpdated,
		&course.UserID,
		&course.InstructorID,
	)
	if err != nil {
		return nil, err
	}
	return &course, nil
}

func GetCourseByID(db *sql.DB, courseID uuid.UUID) (*Course, error) {
	var course Course
	query := `
        SELECT id, name, semester_term, credit_hours, subject_code, course_id, 
		semester_year, date_created, date_updated, user_id, instructor_id
        FROM webapp.courses
        WHERE id = $1
    `
	err := db.QueryRow(query, courseID).Scan(
		&course.ID,
		&course.Name,
		&course.SemesterTerm,
		&course.CreditHours,
		&course.SubjectCode,
		&course.CourseID,
		&course.SemesterYear,
		&course.DateCreated,
		&course.DateUpdated,
		&course.UserID,
		&course.InstructorID,
	)
	if err != nil {
		return nil, err
	}
	return &course, nil
}

// UpdateCourse updates a course, always setting user_id to the authenticated user's ID.
func UpdateCourse(db *sql.DB, courseID uuid.UUID, req UpdateCourseRequest, userID uuid.UUID) (*Course, error) {
	var setClauses []string
	var args []interface{}
	argIndex := 1

	// Always set user_id to the authenticated user's ID
	setClauses = append(setClauses, fmt.Sprintf("user_id = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	// Include optional fields from the request if provided
	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.SemesterTerm != nil {
		setClauses = append(setClauses, fmt.Sprintf("semester_term = $%d", argIndex))
		args = append(args, *req.SemesterTerm)
		argIndex++
	}
	if req.CreditHours != nil {
		setClauses = append(setClauses, fmt.Sprintf("credit_hours = $%d", argIndex))
		args = append(args, *req.CreditHours)
		argIndex++
	}
	if req.SubjectCode != nil {
		setClauses = append(setClauses, fmt.Sprintf("subject_code = $%d", argIndex))
		args = append(args, *req.SubjectCode)
		argIndex++
	}
	if req.CourseID != nil {
		setClauses = append(setClauses, fmt.Sprintf("course_id = $%d", argIndex))
		args = append(args, *req.CourseID)
		argIndex++
	}
	if req.SemesterYear != nil {
		setClauses = append(setClauses, fmt.Sprintf("semester_year = $%d", argIndex))
		args = append(args, *req.SemesterYear)
		argIndex++
	}
	if req.InstructorID != nil {
		setClauses = append(setClauses, fmt.Sprintf("instructor_id = $%d", argIndex))
		args = append(args, *req.InstructorID)
		argIndex++
	}

	// Always update date_updated to the current timestamp
	setClauses = append(setClauses, "date_updated = CURRENT_TIMESTAMP")

	// Construct the SQL query
	query := "UPDATE webapp.courses SET " + strings.Join(setClauses, ", ") +
		fmt.Sprintf(" WHERE id = $%d RETURNING id, name, semester_term, credit_hours, subject_code, course_id, semester_year, date_created, date_updated, user_id, instructor_id", argIndex)
	args = append(args, courseID)

	// Execute the query and scan the result
	var course Course
	err := db.QueryRow(query, args...).Scan(
		&course.ID,
		&course.Name,
		&course.SemesterTerm,
		&course.CreditHours,
		&course.SubjectCode,
		&course.CourseID,
		&course.SemesterYear,
		&course.DateCreated,
		&course.DateUpdated,
		&course.UserID,
		&course.InstructorID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("course not found")
		}
		return nil, err
	}
	return &course, nil
}

func DeleteCourseByID(db *sql.DB, courseID uuid.UUID) error {
	query := "DELETE FROM webapp.courses WHERE id = $1"
	result, err := db.Exec(query, courseID)
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

func InsertTrace(db *sql.DB, userID, instructorID uuid.UUID, status string, courseID uuid.UUID, vectorID *string, fileName, bucketURL string) error {
	query := `
        INSERT INTO webapp.traces (user_id, instructor_id, status, course_id, vector_id, file_name, bucket_url)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `
	_, err := db.Exec(query, userID, instructorID, status, courseID, vectorID, fileName, bucketURL)
	if err != nil {
		log.Printf("Database error: %v", err)
	}
	return err
}

func GetTracesByCourseID(db *sql.DB, courseID uuid.UUID) ([]Trace, error) {
	query := `
        SELECT id, user_id, instructor_id, course_id, status, vector_id, file_name, bucket_url, date_created, date_updated
        FROM webapp.traces
        WHERE course_id = $1
        ORDER BY date_created DESC
    `

	rows, err := db.Query(query, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []Trace
	for rows.Next() {
		var trace Trace
		var vectorID sql.NullString

		err := rows.Scan(
			&trace.ID,
			&trace.UserID,
			&trace.InstructorID,
			&courseID,
			&trace.Status,
			&vectorID,
			&trace.FileName,
			&trace.BucketURL,
			&trace.DateCreated,
			&trace.DateUpdated,
		)
		if err != nil {
			return nil, err
		}

		if vectorID.Valid {
			vectorIDStr := vectorID.String
			trace.VectorID = &vectorIDStr
		}

		traces = append(traces, trace)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return traces, nil
}

func GetTraceByID(db *sql.DB, courseID, traceID uuid.UUID) (*Trace, error) {
	query := `
		SELECT id, user_id, instructor_id, status, vector_id, file_name, bucket_url, date_created, date_updated
		FROM webapp.traces
		WHERE course_id = $1 AND id = $2
	`

	var trace Trace
	var vectorID sql.NullString

	err := db.QueryRow(query, courseID, traceID).Scan(
		&trace.ID,
		&trace.UserID,
		&trace.InstructorID,
		&trace.Status,
		&vectorID,
		&trace.FileName,
		&trace.BucketURL,
		&trace.DateCreated,
		&trace.DateUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	if vectorID.Valid {
		vectorIDStr := vectorID.String
		trace.VectorID = &vectorIDStr
	}

	return &trace, nil
}

func DeleteTraceByID(db *sql.DB, courseID, traceID uuid.UUID) error {
	query := `
		DELETE FROM webapp.traces
		WHERE course_id = $1 AND id = $2
	`

	result, err := db.Exec(query, courseID, traceID)
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
