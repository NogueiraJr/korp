package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"korp/estoque/internal/models"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
)

type ProductRepository struct {
	Pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{Pool: pool}
}

func (r *ProductRepository) Create(ctx context.Context, p *models.Product) error {
	err := r.Pool.QueryRow(ctx,
		`INSERT INTO products (code, description, stock_quantity)
		 VALUES ($1, $2, $3)
		 RETURNING created_at, updated_at`,
		p.Code, p.Description, p.StockQuantity,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
	return err
}

func (r *ProductRepository) List(ctx context.Context) ([]models.Product, error) {
	rows, err := r.Pool.Query(ctx,
		`SELECT code, description, stock_quantity, created_at, updated_at
		 FROM products ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.Code, &p.Description, &p.StockQuantity, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *ProductRepository) GetByCode(ctx context.Context, code string) (*models.Product, error) {
	var p models.Product
	err := r.Pool.QueryRow(ctx,
		`SELECT code, description, stock_quantity, created_at, updated_at
		 FROM products WHERE code = $1`, code).
		Scan(&p.Code, &p.Description, &p.StockQuantity, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) Update(ctx context.Context, code string, desc string, qty int) error {
	tag, err := r.Pool.Exec(ctx,
		`UPDATE products SET description = $1, stock_quantity = $2, updated_at = NOW()
		 WHERE code = $3`, desc, qty, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

// ConsumeStock deducts quantities from multiple products inside a single
// transaction using a guarded UPDATE (WHERE stock_quantity >= qty). Because the
// UPDATE and the row lock are atomic at the database level, two concurrent
// requests can never oversell the same stock.
func (r *ProductRepository) ConsumeStock(ctx context.Context, items []models.ConsumeItem) error {
	if len(items) == 0 {
		return errors.New("no items to consume")
	}

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		if item.Quantity <= 0 {
			return fmt.Errorf("invalid quantity %d for product %s", item.Quantity, item.Code)
		}
		tag, err := tx.Exec(ctx,
			`UPDATE products
			 SET stock_quantity = stock_quantity - $1, updated_at = NOW()
			 WHERE code = $2 AND stock_quantity >= $1`,
			item.Quantity, item.Code)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrInsufficientStock
		}
	}

	return tx.Commit(ctx)
}
