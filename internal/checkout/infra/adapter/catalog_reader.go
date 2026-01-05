package adapter

import (
	"context"

	catalogapp "github.com/dwikikusuma/shoping-llm/internal/catalog/app"
	checkoutapp "github.com/dwikikusuma/shoping-llm/internal/checkout/app"
)

type CatalogServiceReader struct {
	svc *catalogapp.Service
}

func NewCatalogServiceReader(svc *catalogapp.Service) *CatalogServiceReader {
	return &CatalogServiceReader{svc: svc}
}

func (r *CatalogServiceReader) GetProduct(ctx context.Context, productID string) (checkoutapp.Product, error) {
	p, err := r.svc.GetProduct(ctx, productID)
	if err != nil {
		return checkoutapp.Product{}, err
	}

	return checkoutapp.Product{
		ID:       p.ID,
		Name:     p.Name,
		Currency: p.Price.Currency,
		Amount:   p.Price.Amount,
	}, nil
}
