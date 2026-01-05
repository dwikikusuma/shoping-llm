package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/dwikikusuma/shoping-llm/internal/checkout/domain"
	"golang.org/x/sync/errgroup"
)

type CartReader interface {
	GetCart(ctx context.Context, userID string) ([]CartItem, error)
}

type CartItem struct {
	ProductID string
	Quantity  int64
}
type CatalogReader interface {
	GetProduct(ctx context.Context, productID string) (Product, error)
}

type Product struct {
	ID       string
	Name     string
	Currency string
	Amount   int64
}

type Service struct {
	Cart    CartReader
	Catalog CatalogReader

	maxConcurrent int
}

func NewService(cart CartReader, catalog CatalogReader, maxConcurrent int) *Service {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	return &Service{
		Cart:          cart,
		Catalog:       catalog,
		maxConcurrent: maxConcurrent,
	}
}

var ErrEmptyCart = errors.New("cart is empty")

func (s *Service) Quote(ctx context.Context, userID string) (domain.Quote, error) {
	items, err := s.Cart.GetCart(ctx, userID)
	if err != nil {
		return domain.Quote{}, err
	}

	if len(items) == 0 {
		return domain.Quote{}, ErrEmptyCart
	}

	lines := make([]domain.QuoteLine, len(items))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.maxConcurrent)

	for idx := range items {
		idx := idx
		g.Go(func() error {
			it := items[idx]
			if it.Quantity <= 0 {
				return fmt.Errorf("quantity must be greater than zero: %d", it.Quantity)
			}

			product, err := s.Catalog.GetProduct(ctx, it.ProductID)
			if err != nil {
				return fmt.Errorf("failed to get product %s: %w", it.ProductID, err)
			}

			lineTotal := product.Amount * it.Quantity
			lines[idx] = domain.QuoteLine{
				ProductID: product.ID,
				Name:      product.Name,
				Quantity:  it.Quantity,
				UnitPrice: domain.Money{
					Currency: product.Currency,
					Amount:   product.Amount,
				},
				LineTotal: domain.Money{
					Currency: product.Currency,
					Amount:   lineTotal,
				},
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return domain.Quote{}, err
	}

	var totalAmount int64
	for _, line := range lines {
		totalAmount += line.LineTotal.Amount
	}

	quote := domain.Quote{
		Lines: lines,
		Total: domain.Money{
			Currency: lines[0].LineTotal.Currency,
			Amount:   totalAmount,
		},
	}

	return quote, nil
}
