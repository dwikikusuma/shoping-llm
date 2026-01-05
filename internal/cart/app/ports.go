package app

import (
	"context"

	"github.com/dwikikusuma/shoping-llm/internal/cart/domain"
)

type CartRepo interface {
	Get(ctx context.Context, userID string) (domain.Cart, error)
	Create(ctx context.Context, cart domain.Cart) (domain.Cart, error)
	AddItem(ctx context.Context, item domain.CartItem, cartId string) error
	ClearCart(ctx context.Context, cartId string) error
	RemoveItem(ctx context.Context, cartID string, productID string) error
	SetItemQuantity(ctx context.Context, cartID string, item domain.CartItem) error
	GetOrCreate(ctx context.Context, userID string) (domain.Cart, error)
}
