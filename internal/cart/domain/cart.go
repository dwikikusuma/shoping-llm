package domain

import "time"

type CartItem struct {
	ProductID string
	Quantity  int32
}

type Cart struct {
	ID        string
	UserID    string
	Status    string
	Items     []CartItem
	CreatedAt time.Time
	UpdatedAt time.Time
}
