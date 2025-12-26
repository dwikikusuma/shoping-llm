package domain

import "time"

type Money struct {
	Currency string
	Amount   int64
}

type Product struct {
	ID          string
	Name        string
	Price       Money
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
