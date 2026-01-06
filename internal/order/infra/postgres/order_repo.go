package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dwikikusuma/shoping-llm/internal/order/domain"
	"github.com/dwikikusuma/shoping-llm/internal/order/infra/postgres/orderdb"
	"github.com/google/uuid"
)

type OrderRepo struct {
	*orderdb.Queries
	db *sql.DB
}

func NewOrderRepo(db *sql.DB) *OrderRepo {
	return &OrderRepo{
		Queries: orderdb.New(db),
		db:      db,
	}
}

func (r *OrderRepo) execTX(ctx context.Context, fn func(queries *orderdb.Queries) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := orderdb.New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %w; rollback err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

func (r *OrderRepo) CreateOrderTx(ctx context.Context, order domain.Order) (domain.Order, error) {
	var createdOrder domain.Order

	err := r.execTX(ctx, func(q *orderdb.Queries) error {
		o, err := q.CreateOrder(ctx, orderdb.CreateOrderParams{
			UserID:         order.UserID,
			Status:         order.Status,
			Currency:       order.Currency,
			SubtotalAmount: order.SubTotalAmount,
			ShippingAmount: order.ShippingAmount,
			TotalAmount:    order.TotalAmount,
		})
		if err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		orderItems := make([]domain.OrderItem, 0, len(order.OrderItems))

		for i, item := range order.OrderItems {
			expected := item.UnitAmount * int64(item.Quantity)
			if item.LineTotalAmount != expected {
				return fmt.Errorf("item %d: line total mismatch", i)
			}

			pUUID, err := uuid.Parse(item.ProductID)
			if err != nil {
				return fmt.Errorf("item %d: invalid product UUID: %w", i, err)
			}

			row, err := q.AddOrderItem(ctx, orderdb.AddOrderItemParams{
				OrderID:         o.ID,
				ProductID:       pUUID,
				Name:            item.Name,
				UnitAmount:      item.UnitAmount,
				Quantity:        item.Quantity,
				LineTotalAmount: item.LineTotalAmount, // Already calculated from service
			})

			if err != nil {
				return fmt.Errorf("failed to insert item %d: %w", i, err)
			}

			orderItems = append(orderItems, domain.OrderItem{
				ID:              row.ID.String(),
				OrderID:         row.OrderID.String(),
				ProductID:       row.ProductID.String(),
				Name:            row.Name,
				UnitAmount:      row.UnitAmount,
				Quantity:        row.Quantity,
				LineTotalAmount: row.LineTotalAmount,
			})
		}

		createdOrder = domain.Order{
			ID:             o.ID.String(),
			UserID:         o.UserID,
			Status:         o.Status,
			Currency:       o.Currency,
			SubTotalAmount: o.SubtotalAmount,
			ShippingAmount: o.ShippingAmount,
			TotalAmount:    o.TotalAmount,
			OrderItems:     orderItems,
			CreatedAt:      o.CreatedAt,
			UpdatedAt:      o.UpdatedAt,
		}

		return nil
	})
	if err != nil {
		return domain.Order{}, err
	}
	return createdOrder, nil
}
