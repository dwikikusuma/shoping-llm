package app_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/dwikikusuma/shoping-llm/internal/cart/app"
	"github.com/dwikikusuma/shoping-llm/internal/cart/domain"
	"github.com/dwikikusuma/shoping-llm/internal/cart/infra/postgres"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// TODO: Replace this with your existing B1.3 DB helper.
// It should return a *sql.DB connected to your test Postgres with migrations applied.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// Example only:
	// db := testutil.OpenDB(t)
	// return db
	t.Fatal("openTestDB not implemented: reuse your B1.3 db test helper")
	return nil
}

func newTestService(t *testing.T) *app.Service {
	t.Helper()
	db := openTestDB(t)
	repo := postgres.NewCartRepo(db)
	return app.NewService(repo)
}

func TestCart_ConcurrentGetOrCreate_SingleActiveCart(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	userID := uuid.NewString()

	const N = 50
	ids := make(map[string]struct{})
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < N; i++ {
		g.Go(func() error {
			cart, err := svc.GetOrCreate(ctx, userID)
			if err != nil {
				return err
			}
			mu.Lock()
			ids[cart.ID] = struct{}{}
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatalf("concurrent GetOrCreate failed: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected exactly 1 active cart id, got %d: %+v", len(ids), ids)
	}
}

func TestCart_ConcurrentAddItemIncrement(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	userID := uuid.NewString()
	productID := uuid.NewString()

	cart, err := svc.GetOrCreate(ctx, userID)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	const N = 100
	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < N; i++ {
		g.Go(func() error {
			return svc.AddItemToCart(ctx, domain.CartItem{
				ProductID: productID,
				Quantity:  1,
			}, cart.ID)
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatalf("concurrent AddItem failed: %v", err)
	}

	updated, err := svc.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("GetCart failed: %v", err)
	}

	gotQty := 0
	for _, it := range updated.Items {
		if it.ProductID == productID {
			gotQty = int(it.Quantity)
			break
		}
	}
	if gotQty != N {
		t.Fatalf("expected quantity=%d, got=%d", N, gotQty)
	}
}
