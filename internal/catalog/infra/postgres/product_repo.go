package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/dwikikusuma/shoping-llm/internal/catalog/app"
	"github.com/dwikikusuma/shoping-llm/internal/catalog/domain"
	"github.com/dwikikusuma/shoping-llm/internal/catalog/infra/postgres/catalogdb"
	"github.com/google/uuid"
)

type ProductRepo struct {
	q *catalogdb.Queries
}

func NewProductRepo(db *sql.DB) *ProductRepo {
	return &ProductRepo{q: catalogdb.New(db)}
}

func (r *ProductRepo) Create(ctx context.Context, p domain.Product) (domain.Product, error) {
	row, err := r.q.CreateProduct(ctx, catalogdb.CreateProductParams{
		Name:        p.Name,
		Description: p.Description,
		PriceAmount: p.Price.Amount,
		Currency:    p.Price.Currency,
	})
	if err != nil {
		return domain.Product{}, err
	}

	return domain.Product{
		ID:          row.ID.String(),
		Name:        row.Name,
		Description: row.Description,
		Price: domain.Money{
			Amount:   row.PriceAmount,
			Currency: row.Currency,
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *ProductRepo) Get(ctx context.Context, id string) (domain.Product, error) {
	prodID, err := uuid.Parse(id)
	if err != nil {
		return domain.Product{}, err
	}

	product, err := r.q.GetProduct(ctx, prodID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Product{}, app.ErrNotFound
	}
	if err != nil {
		return domain.Product{}, err
	}

	return domain.Product{
		ID:          product.ID.String(),
		Name:        product.Name,
		Description: product.Description,
		Price: domain.Money{
			Amount:   product.PriceAmount,
			Currency: product.Currency,
		},
		CreatedAt: product.CreatedAt,
		UpdatedAt: product.UpdatedAt,
	}, nil
}

func (r *ProductRepo) List(ctx context.Context, query string, limit int, cursor string) ([]domain.Product, string, error) {
	var cur uuid.NullUUID
	if strings.TrimSpace(cursor) != "" {
		uid, err := uuid.Parse(strings.TrimSpace(cursor))
		if err != nil {
			return nil, "", app.ErrInvalidInput
		}
		cur = uuid.NullUUID{UUID: uid, Valid: true}
	}

	rows, err := r.q.ListProducts(ctx, catalogdb.ListProductsParams{
		Query:  strings.TrimSpace(query),
		Limit:  int32(limit),
		Cursor: cur,
	})
	if err != nil {
		return nil, "", err
	}

	out := make([]domain.Product, 0, len(rows))
	var nextCursor string

	for _, row := range rows {
		out = append(out, domain.Product{
			ID:          row.ID.String(),
			Name:        row.Name,
			Description: row.Description,
			Price:       domain.Money{Currency: row.Currency, Amount: row.PriceAmount},
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
		nextCursor = row.ID.String()
	}

	if len(out) < limit {
		nextCursor = ""
	}

	return out, nextCursor, nil
}
