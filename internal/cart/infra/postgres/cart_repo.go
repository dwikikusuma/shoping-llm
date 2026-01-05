package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

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

func (r *CartRepo) GetOrCreate(ctx context.Context, userID string) (domain.Cart, error) {
	// 1) Try get
	cart, err := r.Get(ctx, userID)
	if err == nil {
		return cart, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Cart{}, err
	}

	// 2) Not found => try create
	userUUID, parseErr := uuid.Parse(userID)
	if parseErr != nil {
		return domain.Cart{}, parseErr
	}

	_, createErr := r.q.CreateActiveCart(ctx, userUUID)
	if createErr == nil {
		return r.Get(ctx, userID)
	}

	// 3) If someone else created concurrently => re-get
	if isUniqueViolation(createErr) {
		return r.Get(ctx, userID)
	}

	return domain.Cart{}, createErr
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "23505")
}
