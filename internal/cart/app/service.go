package app

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dwikikusuma/shoping-llm/internal/cart/domain"
)

type Service struct {
	repo CartRepo
}

func NewService(repo CartRepo) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) GetCart(ctx context.Context, userID string) (domain.Cart, error) {
	return s.repo.Get(ctx, userID)
}

func (s *Service) CreateCart(ctx context.Context, cart domain.Cart) (domain.Cart, error) {
	return s.repo.Create(ctx, cart)
}

func (s *Service) GetOrCreate(ctx context.Context, userID string) (domain.Cart, error) {
	cart, err := s.repo.Get(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			newCart := domain.Cart{
				UserID: userID,
			}
			return s.repo.Create(ctx, newCart)
		}
	}
	return cart, err
}

func (s *Service) AddItemToCart(ctx context.Context, item domain.CartItem, cartId string) error {
	return s.repo.AddItem(ctx, item, cartId)
}

func (s *Service) ClearCart(ctx context.Context, cartId string) error {
	return s.repo.ClearCart(ctx, cartId)
}

func (s *Service) SetItemQuantity(ctx context.Context, cartID string, item domain.CartItem) error {
	return s.repo.SetItemQuantity(ctx, cartID, item)
}

func (s *Service) RemoveItemFromCart(ctx context.Context, cartID string, productID string) error {
	return s.repo.RemoveItem(ctx, cartID, productID)
}
