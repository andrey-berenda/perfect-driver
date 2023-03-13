package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/models"
)

const orderCreate = `
INSERT INTO orders (user_id, telegram_id) VALUES ($1, $2)
RETURNING id, user_id, source, destination, time, phone, telegram_id;
`

const orderCreateFromLambda = `
INSERT INTO orders (source, destination, time, phone, telegram_id) VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, source, destination, time, phone, telegram_id;
`

const orderGet = `
SELECT id, user_id, source, destination, time, phone, telegram_id
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC;
`

const orderGetByID = `
SELECT id, user_id, source, destination, time, phone, telegram_id
FROM orders
WHERE id = $1
ORDER BY created_at DESC;
`

const orderUpdateField = `
UPDATE orders
SET %s = $2
WHERE id = $1
RETURNING id, user_id, source, destination, time, phone, telegram_id;
`

var ErrNotFound = errors.New("not found")

func (s *Store) OrderCreate(ctx context.Context, userID uuid.UUID, telegramID int64) (*models.Order, error) {
	rows, err := s.conn.Query(ctx, orderCreate, userID, telegramID)
	o := &models.Order{}
	if err != nil {
		return nil, fmt.Errorf("conn.Exec: %w", err)
	}
	rows.Next()

	err = scanOrder(o, rows)
	rows.Close()
	return o, err
}

func (s *Store) OrderCreateFromLambda(
	ctx context.Context,
	source string,
	destination string,
	time string,
	phone string,
) (*models.Order, error) {
	rows, err := s.conn.Query(ctx, orderCreateFromLambda, source, destination, time, phone, 0)
	o := &models.Order{}
	if err != nil {
		return nil, fmt.Errorf("conn.Exec: %w", err)
	}
	rows.Next()

	err = scanOrder(o, rows)
	rows.Close()
	return o, err
}

func (s *Store) OrderGet(ctx context.Context, userID uuid.UUID) (*models.Order, error) {
	rows, err := s.conn.Query(ctx, orderGet, userID)
	o := &models.Order{}
	if err != nil {
		return nil, fmt.Errorf("conn.Exec: %w", err)
	}
	if !rows.Next() {
		return nil, errors.New("not found")
	}

	err = scanOrder(o, rows)
	if err != nil {
		panic(err)
	}
	rows.Close()
	return o, err
}

func (s *Store) OrderGetByID(ctx context.Context, orderID int) (*models.Order, error) {
	rows, err := s.conn.Query(ctx, orderGetByID, orderID)
	o := &models.Order{}
	if err != nil {
		return nil, fmt.Errorf("conn.Exec: %w", err)
	}
	if !rows.Next() {
		return nil, ErrNotFound
	}

	err = scanOrder(o, rows)
	if err != nil {
		panic(err)
	}
	rows.Close()
	return o, err
}

func (s *Store) OrderSetField(ctx context.Context, orderID int, field string, value string) (*models.Order, error) {
	q := fmt.Sprintf(orderUpdateField, field)
	rows, err := s.conn.Query(ctx, q, orderID, value)
	o := &models.Order{}
	if err != nil {
		return nil, fmt.Errorf("conn.Exec: %w", err)
	}
	rows.Next()

	err = scanOrder(o, rows)
	rows.Close()
	return o, err
}

func scanOrder(o *models.Order, rows pgx.Rows) error {
	return rows.Scan(
		&o.ID,
		&o.UserID,
		&o.Source,
		&o.Destination,
		&o.Time,
		&o.Phone,
		&o.TelegramID,
	)
}
