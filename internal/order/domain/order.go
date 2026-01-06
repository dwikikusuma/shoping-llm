package domain

import "time"

type Order struct {
	ID             string
	UserID         string
	Status         string
	Currency       string
	SubTotalAmount int64
	ShippingAmount int64
	TotalAmount    int64
	OrderItems     []OrderItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OrderItem struct {
	ID              string
	OrderID         string
	ProductID       string
	Name            string
	UnitAmount      int64
	Quantity        int32
	LineTotalAmount int64
}

type CreateOrderRequest struct {
	UserID         string
	Currency       string
	ShippingAmount int64
	Items          []OrderItemRequest
}

type OrderItemRequest struct {
	ProductID  string
	Name       string
	UnitAmount int64
	Quantity   int32
}

type OrderResponse struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	TotalAmount int64     `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
}
