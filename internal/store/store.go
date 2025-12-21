package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	DB *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{DB: db}
}

func (s *Store) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := s.DB.Exec(ctx, sql, args...)
	return err
}
