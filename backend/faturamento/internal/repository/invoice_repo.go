package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"korp/faturamento/internal/models"
)

var (
	ErrInvoiceNotFound = errors.New("invoice not found")
	ErrInvoiceClosed   = errors.New("invoice is already closed")
)

type InvoiceRepository struct {
	Pool *pgxpool.Pool
}

func NewInvoiceRepository(pool *pgxpool.Pool) *InvoiceRepository {
	return &InvoiceRepository{Pool: pool}
}

// Create persists an invoice with a gap-free sequential number. The counter is
// updated inside the same transaction, so a failed insert rolls the number
// back too.
func (r *InvoiceRepository) Create(ctx context.Context, req models.CreateInvoiceRequest) (*models.Invoice, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var number int
	if err := tx.QueryRow(ctx,
		`UPDATE invoice_counters SET next_number = next_number + 1 WHERE id = 1 RETURNING next_number`,
	).Scan(&number); err != nil {
		return nil, err
	}

	var invoiceID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO invoices (number) VALUES ($1) RETURNING id`, number,
	).Scan(&invoiceID); err != nil {
		return nil, err
	}

	for _, item := range req.Items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO invoice_items (invoice_id, product_code, quantity) VALUES ($1, $2, $3)`,
			invoiceID, item.ProductCode, item.Quantity); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, invoiceID)
}

func (r *InvoiceRepository) GetByID(ctx context.Context, id string) (*models.Invoice, error) {
	invoice, err := r.scanInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := r.loadItems(ctx, invoice); err != nil {
		return nil, err
	}
	return invoice, nil
}

func (r *InvoiceRepository) List(ctx context.Context) ([]models.Invoice, error) {
	rows, err := r.Pool.Query(ctx,
		`SELECT id, number, status, created_at, closed_at FROM invoices ORDER BY number DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	var order []string
	invoices := map[string]*models.Invoice{}
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(&inv.ID, &inv.Number, &inv.Status, &inv.CreatedAt, &inv.ClosedAt); err != nil {
			return nil, err
		}
		invoices[inv.ID] = &inv
		ids = append(ids, inv.ID)
		order = append(order, inv.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) > 0 {
		itemRows, err := r.Pool.Query(ctx,
			`SELECT id, invoice_id, product_code, quantity FROM invoice_items WHERE invoice_id = ANY($1) ORDER BY id`,
			ids)
		if err != nil {
			return nil, err
		}
		defer itemRows.Close()
		for itemRows.Next() {
			var item models.InvoiceItem
			var invoiceID string
			if err := itemRows.Scan(&item.ID, &invoiceID, &item.ProductCode, &item.Quantity); err != nil {
				return nil, err
			}
			if inv, ok := invoices[invoiceID]; ok {
				inv.Items = append(inv.Items, item)
			}
		}
		if err := itemRows.Err(); err != nil {
			return nil, err
		}
	}

	result := make([]models.Invoice, 0, len(order))
	for _, id := range order {
		result = append(result, *invoices[id])
	}
	return result, nil
}

func (r *InvoiceRepository) scanInvoice(ctx context.Context, id string) (*models.Invoice, error) {
	var inv models.Invoice
	err := r.Pool.QueryRow(ctx,
		`SELECT id, number, status, created_at, closed_at FROM invoices WHERE id = $1`, id).
		Scan(&inv.ID, &inv.Number, &inv.Status, &inv.CreatedAt, &inv.ClosedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *InvoiceRepository) loadItems(ctx context.Context, inv *models.Invoice) error {
	rows, err := r.Pool.Query(ctx,
		`SELECT id, product_code, quantity FROM invoice_items WHERE invoice_id = $1 ORDER BY id`, inv.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.InvoiceItem
		if err := rows.Scan(&item.ID, &item.ProductCode, &item.Quantity); err != nil {
			return err
		}
		inv.Items = append(inv.Items, item)
	}
	return rows.Err()
}

// ClaimClose atomically transitions an OPEN invoice to CLOSED. Returns
// ErrInvoiceClosed when the invoice is already closed (guards against
// concurrent prints of the same invoice).
func (r *InvoiceRepository) ClaimClose(ctx context.Context, id string) error {
	tag, err := r.Pool.Exec(ctx,
		`UPDATE invoices SET status = 'CLOSED', closed_at = NOW() WHERE id = $1 AND status = 'OPEN'`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInvoiceClosed
	}
	return nil
}

// RevertClose returns a claimed invoice back to OPEN when the stock
// consumption fails, so the user can retry.
func (r *InvoiceRepository) RevertClose(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx,
		`UPDATE invoices SET status = 'OPEN', closed_at = NULL WHERE id = $1`, id)
	return err
}

func (r *InvoiceRepository) GetIdempotencyResponse(ctx context.Context, key string) ([]byte, bool, error) {
	var response []byte
	err := r.Pool.QueryRow(ctx,
		`SELECT response FROM idempotency_keys WHERE key = $1`, key).Scan(&response)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return response, true, nil
}

func (r *InvoiceRepository) SaveIdempotencyResponse(ctx context.Context, key string, response []byte) error {
	_, err := r.Pool.Exec(ctx,
		`INSERT INTO idempotency_keys (key, response) VALUES ($1, $2) ON CONFLICT (key) DO NOTHING`,
		key, response)
	return err
}