package repository

import (
	"context"
	"fmt"
	"rinha-go/entities"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	conn *pgxpool.Pool
}

func NewRepository(conn *pgxpool.Pool) (*Repository, error) {

	if conn == nil {
		return nil, fmt.Errorf("connection cannot be nil")
	}

	return &Repository{
		conn: conn,
	}, nil
}

func (r *Repository) CreateTables() error {
	query := `CREATE TABLE IF NOT EXISTS payments_to_send (
			    id           TEXT PRIMARY KEY,
			    amount       DOUBLE PRECISION NOT NULL,
			    requested_at TIMESTAMP WITH TIME ZONE NOT NULL,
			    is_default   BOOLEAN NOT NULL DEFAULT false
			);`

	_, err := r.conn.Exec(context.Background(), query)
	if err != nil {
		return fmt.Errorf("error creating payments_to_send table: %v", err)
	}
	return nil
}

func (r *Repository) GetPaymentsSummary(from *string, to *string) (*entities.PaymentSummary, error) {
	if r.conn == nil {
		return nil, fmt.Errorf("db is not initialized")
	}

	query := `
		SELECT is_default::int, COUNT(*) as total_requests, COALESCE(SUM(amount), 0) as total_amount
		FROM payments_to_send
		WHERE 1=1
	`
	args := []interface{}{}

	if from != nil {
		query += " AND requested_at >= $1"
		args = append(args, *from)
	}

	if to != nil {
		query += " AND requested_at <= $2"
		args = append(args, *to)
	}

	query += " GROUP BY is_default"

	rows, err := r.conn.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := entities.PaymentSummary{}

	for rows.Next() {
		var isDefault int
		var totalRequests int
		var totalAmount float64

		err := rows.Scan(&isDefault, &totalRequests, &totalAmount)
		if err != nil {
			return nil, err
		}

		if isDefault == 1 {
			summary.Default.TotalRequests = totalRequests
			summary.Default.TotalAmount = totalAmount
		} else {
			summary.Fallback.TotalRequests = totalRequests
			summary.Fallback.TotalAmount = totalAmount
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &summary, nil
}

func (r *Repository) SavePayment(payment *entities.PaymentToSend, isDefault bool) error {
	if r.conn == nil {
		return fmt.Errorf("db is not initialized")
	}

	// Validar timestamp
	t, err := time.Parse(time.RFC3339, payment.RequestedAt)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %v", payment.RequestedAt)
	}

	// Montar insert
	query := `
		INSERT INTO payments_to_send (id, amount, requested_at, is_default)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE 
		SET amount = EXCLUDED.amount,
		    requested_at = EXCLUDED.requested_at,
		    is_default = EXCLUDED.is_default
	`

	_, err = r.conn.Exec(context.Background(), query, payment.ID, payment.Amount, t, isDefault)
	if err != nil {
		return fmt.Errorf("failed to insert payment: %w", err)
	}

	return nil
}
