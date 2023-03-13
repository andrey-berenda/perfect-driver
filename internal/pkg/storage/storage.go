package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

func New() *Store {
	databaseURL := "postgresql://driveruser:de3d8207-7077-44e4-a2f8-2efdfe085f51@db.c5cmlmm4rkvj.us-east-1.rds.amazonaws.com:5432/postgres"
	pool, err := pgxpool.Connect(context.Background(), databaseURL)
	if err != nil {
		panic(fmt.Sprintf("pgxpool.Connect: %s", err))
	}
	return &Store{conn: pool}
}

type Store struct {
	conn *pgxpool.Pool
}
