package app

import (
	"context"

	"github.com/dwikikusuma/shoping-llm/internal/order/domain"
)

type OrderRepo interface {
	CreateOrderTx(ctx context.Context, order domain.Order) (domain.Order, error)
}
