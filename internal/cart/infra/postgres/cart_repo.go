package postgres

import (
	"context"
	"database/sql"

	"github.com/dwikikusuma/shoping-llm/internal/cart/domain"
	"github.com/dwikikusuma/shoping-llm/internal/cart/infra/postgres/cartgdb"
	"github.com/google/uuid"
)

type CartRepo struct {
	q *cartgdb.Queries
}

func NewCartRepo(db *sql.DB) *CartRepo {
	return &CartRepo{
		q: cartgdb.New(db),
	}
}

func (r *CartRepo) Get(ctx context.Context, userID string) (domain.Cart, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return domain.Cart{}, err
	}

	cart, err := r.q.GetActiveCartByUserID(ctx, userUUID)
	if err != nil {
		return domain.Cart{}, err
	}

	cartItem, err := r.q.ListCartItems(ctx, cart.ID)
	if err != nil {
		return domain.Cart{}, err
	}

	var items []domain.CartItem
	for _, item := range cartItem {
		items = append(items, domain.CartItem{
			ProductID: item.ProductID.String(),
			Quantity:  item.Quantity,
		})
	}

	return domain.Cart{
		ID:        cart.ID.String(),
		UserID:    cart.UserID.String(),
		Status:    cart.Status,
		Items:     items,
		CreatedAt: cart.CreatedAt,
		UpdatedAt: cart.UpdatedAt,
	}, nil
}

func (r *CartRepo) Create(ctx context.Context, cart domain.Cart) (domain.Cart, error) {
	userUUID, err := uuid.Parse(cart.UserID)
	if err != nil {
		return domain.Cart{}, err
	}

	newCart, err := r.q.CreateActiveCart(ctx, userUUID)
	if err != nil {
		return domain.Cart{}, err
	}

	if cart.Items != nil && len(cart.Items) > 0 {
		for _, item := range cart.Items {
			err = r.AddItem(ctx, item, newCart.ID.String())
			if err != nil {
				return domain.Cart{}, err
			}
		}
	}

	return r.Get(ctx, cart.UserID)
}

func (r *CartRepo) AddItem(ctx context.Context, item domain.CartItem, cartId string) error {
	cartUUID, err := uuid.Parse(cartId)
	if err != nil {
		return err
	}

	productUUID, err := uuid.Parse(item.ProductID)
	if err != nil {
		return err
	}

	_, err = r.q.UpsertAddItemIncrement(ctx, cartgdb.UpsertAddItemIncrementParams{
		CartID:    cartUUID,
		ProductID: productUUID,
		Quantity:  item.Quantity,
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *CartRepo) ClearCart(ctx context.Context, cartId string) error {
	cartUUID, err := uuid.Parse(cartId)
	if err != nil {
		return err
	}

	err = r.q.ClearCart(ctx, cartUUID)
	if err != nil {
		return err
	}

	return nil
}

func (r *CartRepo) RemoveItem(ctx context.Context, cartID string, productID string) error {
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return err
	}

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		return err
	}

	err = r.q.RemoveItem(ctx, cartgdb.RemoveItemParams{
		CartID:    cartUUID,
		ProductID: productUUID,
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *CartRepo) SetItemQuantity(ctx context.Context, cartID string, item domain.CartItem) error {
	cartUUID, err := uuid.Parse(cartID)
	if err != nil {
		return err
	}

	productUUID, err := uuid.Parse(item.ProductID)
	if err != nil {
		return err
	}

	_, err = r.q.SetItemQuantity(ctx, cartgdb.SetItemQuantityParams{
		CartID:    cartUUID,
		ProductID: productUUID,
		Quantity:  item.Quantity,
	})

	if err != nil {
		return err
	}

	return nil
}
