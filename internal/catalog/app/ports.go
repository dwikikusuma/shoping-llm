package app

import (
	"context"

	"github.com/dwikikusuma/shoping-llm/internal/catalog/domain"
)

type ProductRepo interface {
	Create(ctx context.Context, p domain.Product) (domain.Product, error)
	Get(ctx context.Context, id string) (domain.Product, error)
	List(ctx context.Context, query string, limit int, cursor string) ([]domain.Product, string, error)
}
