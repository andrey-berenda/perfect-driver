package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/models"
)

const upsertUser = `
INSERT INTO users (telegram_id) VALUES ($1)
ON CONFLICT (telegram_id) DO UPDATE SET telegram_id = $1
RETURNING id, telegram_id;
`

const selectUser = `
SELECT id, telegram_id
FROM users
WHERE id = $1;
`

func (s *Store) UserGet(ctx context.Context, telegramID int64) (*models.User, error) {
	rows, err := s.conn.Query(ctx, upsertUser, telegramID)
	u := &models.User{}
	if err != nil {
		return nil, fmt.Errorf("conn.Exec: %w", err)
	}
	rows.Next()

	err = scanUser(u, rows)
	rows.Close()
	return u, err
}

func (s *Store) UserGetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	rows, err := s.conn.Query(ctx, selectUser, userID)
	u := &models.User{}
	if err != nil {
		return nil, fmt.Errorf("conn.Exec: %w", err)
	}
	defer rows.Close()
	rows.Next()

	err = scanUser(u, rows)
	return u, err
}

func scanUser(u *models.User, rows pgx.Rows) error {
	return rows.Scan(
		&u.ID,
		&u.TelegramID,
	)
}
