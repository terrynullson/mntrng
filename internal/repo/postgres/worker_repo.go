package postgres

import "database/sql"

type WorkerRepo struct {
	db *sql.DB
}

func NewWorkerRepo(db *sql.DB) *WorkerRepo {
	return &WorkerRepo{db: db}
}
