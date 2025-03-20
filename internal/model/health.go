// internal/model/health.go
package model

import (
	"database/sql"
	"time"
)

type HealthCheck struct {
	CheckID  int64     `json:"check_id"`
	DateTime time.Time `json:"datetime"`
}

func InsertHealthCheck(db *sql.DB) error {
	// PostgreSQL uses CURRENT_TIMESTAMP instead of UTC_TIMESTAMP()
	query := "INSERT INTO webapp.health_check (datetime) VALUES (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')"
	_, err := db.Exec(query)
	return err
}
