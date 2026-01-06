package app

import (
	"context"
	"fmt"

	"github.com/dwikikusuma/shoping-llm/internal/order/domain"
)

type Service struct {
	repo OrderRepo
}

const (
	OrderStatusPending = "PENDING"
)

func NewService(repo OrderRepo) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateOrder(ctx context.Context, req domain.CreateOrderRequest) (domain.OrderResponse, error) {
	if req.ShippingAmount < 0 {
		return domain.OrderResponse{}, fmt.Errorf("shipping amount cannot be negative, got %d", req.ShippingAmount)
	}

	orderItem := make([]domain.OrderItem, 0, len(req.Items))
	var subTotalAmount int64 = 0

	for i, item := range req.Items {
		if item.Quantity <= 0 {
			return domain.OrderResponse{}, fmt.Errorf("item %d: quantity must be positive, got %d", i, item.Quantity)
		}
		if item.UnitAmount < 0 {
			return domain.OrderResponse{}, fmt.Errorf("item %d: unit amount cannot be negative, got %d", i, item.UnitAmount)
		}

		orderItem = append(orderItem, domain.OrderItem{
			ProductID:       item.ProductID,
			Name:            item.Name,
			UnitAmount:      item.UnitAmount,
			Quantity:        item.Quantity,
			LineTotalAmount: item.UnitAmount * int64(item.Quantity),
		})

		subTotalAmount += item.UnitAmount * int64(item.Quantity)
	}

	order := domain.Order{
		UserID:         req.UserID,
		Status:         OrderStatusPending,
		Currency:       req.Currency,
		ShippingAmount: req.ShippingAmount,
		SubTotalAmount: subTotalAmount,
		TotalAmount:    subTotalAmount + req.ShippingAmount,
		OrderItems:     orderItem,
	}

	createdOrder, err := s.repo.CreateOrderTx(ctx, order)
	if err != nil {
		return domain.OrderResponse{}, err
	}

	return domain.OrderResponse{
		ID:          createdOrder.ID,
		Status:      createdOrder.Status,
		TotalAmount: createdOrder.TotalAmount,
		CreatedAt:   createdOrder.CreatedAt,
	}, nil
}
