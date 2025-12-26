package app

import (
	"context"
	"errors"
	"strings"

	"github.com/dwikikusuma/shoping-llm/internal/catalog/domain"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
)

type Service struct {
	repo ProductRepo
}

func NewService(repo ProductRepo) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateProduct(ctx context.Context, name, desc, currency string, amount int64) (domain.Product, error) {
	name = strings.TrimSpace(name)
	currency = strings.TrimSpace(currency)

	if name == "" || currency == "" || amount <= 0 {
		return domain.Product{}, ErrInvalidInput
	}

	p := domain.Product{
		Name:        name,
		Description: desc,
		Price: domain.Money{
			Currency: currency,
			Amount:   amount,
		},
	}

	product, err := s.repo.Create(ctx, p)
	if err != nil {
		return domain.Product{}, err
	}

	return product, nil
}

func (s *Service) GetProduct(ctx context.Context, id string) (domain.Product, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Product{}, ErrInvalidInput
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) ListProducts(ctx context.Context, query string, limit int, cursor string) ([]domain.Product, string, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.List(ctx, query, limit, cursor)
}
