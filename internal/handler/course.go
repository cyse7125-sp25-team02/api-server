// internal/handler/course.go
package handler

import (
	"api-server/internal/config"
	"api-server/internal/model"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type CourseHandler struct {
	db         *sql.DB
	gcsClient  *storage.Client
	bucketName string
	producer   sarama.SyncProducer
}

func NewCourseHandler(db *sql.DB, cfg *config.Config, producer sarama.SyncProducer) *CourseHandler {
	ctx := context.Background()
	var client *storage.Client
	var err error

	log.Printf("GCSCredentialsFile: %q", cfg.GCSCredentialsFile)
	if cfg.GCSCredentialsFile != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(cfg.GCSCredentialsFile))
	} else {
		log.Println("Using Application Default Credentials")
		client, err = storage.NewClient(ctx)
	}
	if err != nil {
		log.Fatalf("Failed to create GCS client: %v", err)
	}

	return &CourseHandler{
		db:         db,
		gcsClient:  client,
		bucketName: cfg.GCSBucketName,
		producer:   producer,
	}
}

func (h *CourseHandler) authenticateRequest(w http.ResponseWriter, r *http.Request) (*model.User, error) {
	username, password, hasAuth := r.BasicAuth()
	if !hasAuth {
		return nil, fmt.Errorf("authentication required")
	}

	// Authenticate the user
	user, err := model.AuthenticateUser(h.db, username, password)
	if err != nil {
		return nil, err
	}

	// Check admin privileges
	if user.Role != "admin" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Insufficient permissions"})
		return nil, fmt.Errorf("insufficient permissions")
	}

	return user, nil
}

func (h *CourseHandler) handleAuthError(w http.ResponseWriter, err error) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Course Authentication Required"`)
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func (h *CourseHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Authenticate user
	user, err := h.authenticateRequest(w, r)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	var req model.CreateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate the request data
	if err := req.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Create the course in the database
	course, err := model.CreateCourse(h.db, req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key constraint") {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid instructor_id"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create course"})
		return
	}

	// Return the created course
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(course)
}

func (h *CourseHandler) GetCourseByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Extract the course ID from path parameters
	courseIDStr := r.PathValue("course_id")
	if courseIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Course ID is required"})
		return
	}

	// Parse the course ID into a UUID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course ID format"})
		return
	}

	// Retrieve the course from the database
	course, err := model.GetCourseByID(h.db, courseID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Course not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve course"})
		return
	}

	// Return the course details as JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(course)
}

func (h *CourseHandler) DeleteCourseByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Authenticate user
	_, err := h.authenticateRequest(w, r)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	// Extract the course ID from path parameters
	courseIDStr := r.PathValue("course_id")
	if courseIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Course ID is required"})
		return
	}

	// Parse the course ID as a UUID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course ID format"})
		return
	}

	// Delete the course from the database
	err = model.DeleteCourseByID(h.db, courseID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Course not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete course"})
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Course deleted successfully"})
}

func (h *CourseHandler) PatchCourse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Authenticate user
	user, err := h.authenticateRequest(w, r)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	// Extract the course ID from path parameters
	courseIDStr := r.PathValue("course_id")
	if courseIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Course ID is required"})
		return
	}

	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course ID format"})
		return
	}

	// Parse request body
	var req model.UpdateCourseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Update the course
	updatedCourse, err := model.UpdateCourse(h.db, courseID, req, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key constraint") {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid user_id or instructor_id"})
			return
		}
		if err.Error() == "course not found" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Course not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update course"})
		return
	}

	// Return the updated course
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedCourse)
}

func (h *CourseHandler) HandleTraceUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Authenticate user
	user, err := h.authenticateRequest(w, r)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	// Extract course ID from path parameters
	courseIDStr := r.PathValue("course_id")

	// Parse the course ID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course_id format"})
		return
	}

	// Parse multipart form (max 10MB)
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse multipart form"})
		return
	}

	// Get the PDF file
	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "File is required"})
		return
	}
	defer file.Close()

	var vectorID *string
	if vid := r.FormValue("vector_id"); vid != "" {
		vectorID = &vid
	}

	// Fetch course details
	course, err := model.GetCourseByID(h.db, courseID)
	if err != nil {
		log.Printf("Failed to fetch course: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch course details"})
		return
	}

	// Fetch instructor details
	instructor, err := model.GetInstructorByID(h.db, course.InstructorID)
	if err != nil {
		log.Printf("Failed to fetch instructor: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch instructor details"})
		return
	}

	// Generate custom filename
	customName := fmt.Sprintf(
		"%s_%s_%s_%d_%s_%d.pdf",
		sanitizeFilename(course.Name),
		sanitizeFilename(instructor.Name),
		course.SubjectCode,
		course.CourseID,
		course.SemesterTerm,
		course.SemesterYear,
	)

	// Generate a unique filename for GCS to avoid conflicts
	bucketURL, err := h.uploadToGCS(file, customName)
	status := "uploaded"
	if err != nil {
		log.Printf("GCS upload failed: %v", err)
		status = "failed"
		bucketURL = "" // Since bucket_url is NOT NULL, use empty string
		err = model.InsertTrace(h.db, user.ID, course.InstructorID, status, courseID, vectorID, customName, bucketURL)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to insert trace record"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to upload file to GCS"})
		return
	}

	// Insert trace record on successful upload
	err = model.InsertTrace(h.db, user.ID, course.InstructorID, status, courseID, vectorID, customName, bucketURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to insert trace record"})
		return
	}

	// Produce JSON message to Kafka
	traceMessage := map[string]string{
		"instructor_name": strings.ToLower(instructor.Name),
		"course_code":     strings.ToLower(fmt.Sprintf("%s %d", course.SubjectCode, course.CourseID)),
		"semester_term":   strings.ToLower(course.SemesterTerm),
		"semester_year":   strings.ToLower(fmt.Sprintf("%d", course.SemesterYear)),
		"course_name":     strings.ToLower(course.Name),
		"credit_hours":    strings.ToLower(fmt.Sprintf("%d", course.CreditHours)),
		"bucket_path":     bucketURL,
	}
	messageBytes, err := json.Marshal(traceMessage)
	if err != nil {
		log.Printf("Failed to marshal Kafka message: %v", err)
	} else {
		msg := &sarama.ProducerMessage{
			Topic: "pdf-upload",
			Value: sarama.ByteEncoder(messageBytes),
		}
		partition, offset, err := h.producer.SendMessage(msg)
		if err != nil {
			log.Printf("Failed to send Kafka message: %v", err)
		} else {
			log.Printf("Sent message to partition %d, offset %d", partition, offset)
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully", "bucket_url": bucketURL})
}

func (h *CourseHandler) uploadToGCS(file io.Reader, filename string) (string, error) {
	ctx := context.Background()
	bucket := h.gcsClient.Bucket(h.bucketName)
	object := bucket.Object(filename)

	w := object.NewWriter(ctx)
	if _, err := io.Copy(w, file); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	attrs, err := object.Attrs(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", h.bucketName, attrs.Name), nil
}

func (h *CourseHandler) GetTracesByCourseID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Authenticate user
	_, err := h.authenticateRequest(w, r)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	// Extract course_id from path parameters
	courseIDStr := r.PathValue("course_id")
	// Parse the course ID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course_id format"})
		return
	}

	// Get traces from the database
	traces, err := model.GetTracesByCourseID(h.db, courseID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve traces"})
		return
	}

	// Return the traces as JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"data": traces})
}

func (h *CourseHandler) GetTraceByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Authenticate user
	user, err := h.authenticateRequest(w, r)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	// Check admin privileges
	if user.Role != "admin" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Insufficient permissions"})
		return
	}

	// Extract course_id and trace_id from path parameters
	courseIDStr := r.PathValue("course_id")
	traceIDStr := r.PathValue("trace_id")

	// Parse the course ID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course_id format"})
		return
	}

	// Parse the trace ID
	traceID, err := uuid.Parse(traceIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid trace_id format"})
		return
	}

	// Get trace from the database
	trace, err := model.GetTraceByID(h.db, courseID, traceID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Trace not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve trace"})
		return
	}

	// Return the trace as JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(trace)
}

func (h *CourseHandler) DeleteTraceByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Authenticate user
	_, err := h.authenticateRequest(w, r)
	if err != nil {
		h.handleAuthError(w, err)
		return
	}

	// Extract course_id and trace_id from path parameters
	courseIDStr := r.PathValue("course_id")
	traceIDStr := r.PathValue("trace_id")

	// Parse the course ID
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course_id format"})
		return
	}

	// Parse the trace ID
	traceID, err := uuid.Parse(traceIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid trace_id format"})
		return
	}

	// Delete the trace from the database
	err = model.DeleteTraceByID(h.db, courseID, traceID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Trace not found"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete trace"})
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Trace deleted successfully"})
}

// sanitizeFilename removes spaces and special characters, replacing with underscores or nothing.
func sanitizeFilename(input string) string {
	// Replace spaces and special characters with underscores, keep alphanumeric
	reg, _ := regexp.Compile("[^a-zA-Z0-9]+")
	cleaned := reg.ReplaceAllString(input, "_")
	// Remove leading/trailing underscores
	return strings.Trim(cleaned, "_")
}
