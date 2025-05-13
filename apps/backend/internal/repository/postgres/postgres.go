package postgres

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func NewConnection() (*sql.DB, error) {
	connStr := fmt.Sprintf("postgres://postgres:%s@localhost:5432/postgres?sslmode=disable", os.Getenv("DB_PASS"))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}
