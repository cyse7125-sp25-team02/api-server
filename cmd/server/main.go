// cmd/server/main.go
package main

import (
	"api-server/internal/config"
	"api-server/internal/database"
	"api-server/internal/handler"
	"log"
	"net/http"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.NewConfig()

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	brokers := []string{cfg.KAFKA_BROKER}

	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, kafkaConfig)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka producer: %v", err)
	}
	defer producer.Close()

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Create a custom Prometheus registry to avoid conflicts with default registry
	reg := prometheus.NewRegistry()

	// Register collectors with the custom registry
	if err := reg.Register(collectors.NewGoCollector()); err != nil {
		log.Printf("Failed to register Go collector: %v", err)
	}
	if err := reg.Register(collectors.NewBuildInfoCollector()); err != nil {
		log.Printf("Failed to register BuildInfo collector: %v", err)
	}

	// Define and register the custom counter metric
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests per endpoint",
		},
		[]string{"path", "method"},
	)
	if err := reg.Register(requestCounter); err != nil {
		log.Fatalf("Failed to register requestCounter: %v", err)
	}

	// Middleware to count requests and pass to the handler
	countRequests := func(path string, next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Increment the counter with the path and method
			requestCounter.WithLabelValues(path, r.Method).Inc()
			next.ServeHTTP(w, r)
		})
	}

	// create /healthz endpoint to check if the server is running
	healthHandler := handler.NewHealthHandler(db)
	mux.Handle("/healthz", countRequests("/healthz", healthHandler))

	// User endpoint
	userHandler := handler.NewUserHandler(db)
	mux.Handle("/v1/user", countRequests("/v1/user", userHandler))

	// Instructor endpoint
	instructorHandler := handler.NewInstructorHandler(db)
	mux.Handle("/v1/instructor", countRequests("/v1/instructor", instructorHandler))

	courseHandler := handler.NewCourseHandler(db, cfg, producer)
	mux.Handle("POST /v1/course", countRequests("/v1/course", http.HandlerFunc(courseHandler.CreateCourse)))
	mux.Handle("GET /v1/course/{course_id}", countRequests("/v1/course/{course_id}", http.HandlerFunc(courseHandler.GetCourseByID)))
	mux.Handle("PATCH /v1/course/{course_id}", countRequests("/v1/course/{course_id}", http.HandlerFunc(courseHandler.PatchCourse)))
	mux.Handle("DELETE /v1/course/{course_id}", countRequests("/v1/course/{course_id}", http.HandlerFunc(courseHandler.DeleteCourseByID)))
	mux.Handle("GET /v1/course/{course_id}/trace", countRequests("/v1/course/{course_id}/trace", http.HandlerFunc(courseHandler.GetTracesByCourseID)))
	mux.Handle("POST /v1/course/{course_id}/trace", countRequests("/v1/course/{course_id}/trace", http.HandlerFunc(courseHandler.HandleTraceUpload)))
	mux.Handle("GET /v1/course/{course_id}/trace/{trace_id}", countRequests("/v1/course/{course_id}/trace/{trace_id}", http.HandlerFunc(courseHandler.GetTraceByID)))
	mux.Handle("DELETE /v1/course/{course_id}/trace/{trace_id}", countRequests("/v1/course/{course_id}/trace/{trace_id}", http.HandlerFunc(courseHandler.DeleteTraceByID)))

	// Use the custom registry for the /metrics endpoint
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	log.Println("Server starting on :3000")
	if err := http.ListenAndServe(":3000", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
