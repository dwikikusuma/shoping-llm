package adapter

import (
	"context"

	cartapp "github.com/dwikikusuma/shoping-llm/internal/cart/app"
	checkoutapp "github.com/dwikikusuma/shoping-llm/internal/checkout/app"
)

type CartServiceReader struct {
	svc *cartapp.Service
}

func NewCartServiceReader(svc *cartapp.Service) *CartServiceReader {
	return &CartServiceReader{svc: svc}
}

func (r *CartServiceReader) GetCart(ctx context.Context, userID string) ([]checkoutapp.CartItem, error) {
	cart, err := r.svc.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]checkoutapp.CartItem, 0, len(cart.Items))
	for _, it := range cart.Items {
		items = append(items, checkoutapp.CartItem{
			ProductID: it.ProductID,
			Quantity:  int64(it.Quantity),
		})
	}
	return items, nil
}
