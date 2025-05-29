package repository

import (
	"database/sql"
	"fmt"

	"github.com/kisnikita/safe-disputes/backend/pkg/log"
)

type Repository struct {
	db     *sql.DB
	logger log.Logger
}

func New(db *sql.DB, logger log.Logger) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is nil")
	}
	return &Repository{
		db:     db,
		logger: logger,
	}, nil
}

func (repo *Repository) StartTx() (*sql.Tx, error) {
	return repo.db.Begin()
}
