package app

import (
	"context"
	"testing"

	"github.com/dwikikusuma/shoping-llm/internal/catalog/domain"
)

type fakeRepo struct{}

func (fakeRepo) Create(ctx context.Context, p domain.Product) (domain.Product, error) { return p, nil }
func (fakeRepo) Get(ctx context.Context, id string) (domain.Product, error) {
	return domain.Product{}, nil
}
func (fakeRepo) List(ctx context.Context, query string, limit int, cursor string) ([]domain.Product, string, error) {
	return nil, "", nil
}

func TestCreateProductValidation(t *testing.T) {
	svc := NewService(fakeRepo{})

	t.Run("empty name -> invalid", func(t *testing.T) {
		_, err := svc.CreateProduct(context.Background(), "   ", "x", "IDR", 100)
		if err != ErrInvalidInput {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("negative amount -> invalid", func(t *testing.T) {
		_, err := svc.CreateProduct(context.Background(), "Keyboard", "x", "IDR", -1)
		if err != ErrInvalidInput {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("empty currency -> invalid", func(t *testing.T) {
		_, err := svc.CreateProduct(context.Background(), "Keyboard", "x", "   ", 100)
		if err != ErrInvalidInput {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})
}
