package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"

	"github.com/andrey-berenda/perfect-driver/internal/pkg/models"
)

const insertPayment = `
INSERT INTO payments (
	  id, 
      order_id,
	  status,
	  confirmation_url
) VALUES ($1, $2, $3, $4)
RETURNING id, status, order_id, confirmation_url;
`

const selectPaymentByID = `
SELECT id, status, order_id, confirmation_url
FROM payments
WHERE id = $1;
`

const setPaymentStatusAndPaymentMethodID = `
UPDATE payments
SET status = $2
WHERE id = $1;
`

const selectPendingPayments = `
SELECT id, status, order_id, confirmation_url
FROM payments
WHERE status = $1;
`

func (s *Store) PaymentCreate(ctx context.Context, p models.Payment) error {
	_, err := s.conn.Exec(
		ctx,
		insertPayment,
		p.ID,
		p.OrderID,
		p.Status,
		p.ConfirmationURL,
	)
	if err != nil {
		return fmt.Errorf("conn.Query: %w", err)
	}
	return nil
}

func (s *Store) PaymentSetStatus(ctx context.Context, paymentID uuid.UUID, status models.PaymentStatus) error {
	result, err := s.conn.Exec(
		ctx,
		setPaymentStatusAndPaymentMethodID,
		paymentID,
		status,
	)
	if err != nil {
		return fmt.Errorf("conn.Exec: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) PaymentGet(ctx context.Context, paymentID uuid.UUID) (*models.Payment, error) {
	rows, err := s.conn.Query(
		ctx,
		selectPaymentByID,
		paymentID,
	)
	if err != nil {
		return nil, fmt.Errorf("conn.Query: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, ErrNotFound
	}
	p, err := scanPayment(rows)

	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) PaymentsForCheck(ctx context.Context) ([]models.Payment, error) {
	rows, err := s.conn.Query(
		ctx,
		selectPendingPayments,
		models.PaymentStatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("conn.Query: %w", err)
	}
	defer rows.Close()
	var result []models.Payment

	var p models.Payment
	for rows.Next() {
		p, err = scanPayment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

func (s *Store) PaymentsForCheckChan(ctx context.Context) chan models.Payment {
	ch := make(chan models.Payment, 10)

	go func() {
		ticker := time.NewTicker(time.Second * 10)

		for {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				ticker.Stop()
				close(ch)
				return
			}
			payments, err := s.PaymentsForCheck(ctx)
			if err != nil {
				s.logger.Errorf("PaymentsForCheck: %v", err)
				continue
			}
			for _, p := range payments {
				ch <- p
			}
		}
	}()
	return ch
}

func scanPayment(rows pgx.Rows) (models.Payment, error) {
	p := models.Payment{}
	err := rows.Scan(
		&p.ID,
		&p.Status,
		&p.OrderID,
		&p.ConfirmationURL,
	)
	if err != nil {
		return p, fmt.Errorf("rows.Scan: %w", err)
	}
	return p, nil
}
