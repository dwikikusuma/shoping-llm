package grpc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dwikikusuma/shoping-llm/api/gen/cart/v1"
	"github.com/dwikikusuma/shoping-llm/internal/cart/app"
	"github.com/dwikikusuma/shoping-llm/internal/cart/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	cartv1.UnimplementedCartServiceServer
	svc *app.Service
}

func NewServer(svc *app.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) GetCart(ctx context.Context, req *cartv1.UserId) (*cartv1.Cart, error) {
	cart, err := s.svc.GetCart(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "cart not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "error getting cart: %v", err)
	}

	return toProto(cart), nil
}

func (s *Server) CreateCart(ctx context.Context, req *cartv1.Cart) (*cartv1.Cart, error) {
	var cartItems []domain.CartItem
	for _, item := range req.Items {
		cartItems = append(cartItems, domain.CartItem{
			ProductID: item.ProductId,
			Quantity:  item.Quantity,
		})
	}

	cart := domain.Cart{
		UserID: req.UserId,
		Status: req.Status,
		Items:  cartItems,
	}

	createdCart, err := s.svc.CreateCart(ctx, cart)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error creating cart: %v", err)
	}

	return toProto(createdCart), nil
}

func (s *Server) GetOrCreateCart(ctx context.Context, req *cartv1.UserId) (*cartv1.Cart, error) {
	cart, err := s.svc.GetOrCreate(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting or creating cart: %v", err)
	}
	return toProto(cart), err
}

func (s *Server) AddItem(ctx context.Context, req *cartv1.UpdateCartItemRequest) (*cartv1.Cart, error) {
	cartItem := domain.CartItem{
		ProductID: req.Item.ProductId,
		Quantity:  req.Item.Quantity,
	}

	cart, err := s.svc.GetOrCreate(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting or creating cart: %v", err)
	}

	err = s.svc.AddItemToCart(ctx, cartItem, cart.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error adding item to cart: %v", err)
	}

	updatedCart, err := s.svc.GetCart(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting updated cart: %v", err)
	}

	return toProto(updatedCart), nil
}

func (s *Server) ClearCart(ctx context.Context, req *cartv1.CartId) (*cartv1.Cart, error) {
	cart, err := s.svc.GetCart(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "cart not found: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "error getting cart: %v", err)
	}
	err = s.svc.ClearCart(ctx, cart.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error clearing cart: %v", err)
	}

	clearedCart, err := s.svc.GetCart(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting cleared cart: %v", err)
	}

	return toProto(clearedCart), nil
}

func (s *Server) SetItemQuantity(ctx context.Context, req *cartv1.UpdateCartItemRequest) (*cartv1.Cart, error) {
	cart, err := s.svc.GetCart(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting cart: %v", err)
	}

	cartItem := domain.CartItem{
		ProductID: req.Item.ProductId,
		Quantity:  req.Item.Quantity,
	}

	err = s.svc.SetItemQuantity(ctx, cart.ID, cartItem)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error setting item quantity: %v", err)
	}

	updatedCart, err := s.svc.GetCart(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting updated cart: %v", err)
	}

	return toProto(updatedCart), nil
}

func (s *Server) RemoveItem(ctx context.Context, req *cartv1.RemoveCartItemRequest) (*cartv1.Cart, error) {
	cart, err := s.svc.GetCart(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting cart: %v", err)
	}

	err = s.svc.RemoveItemFromCart(ctx, cart.ID, req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error removing item from cart: %v", err)
	}

	updatedCart, err := s.svc.GetCart(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting updated cart: %v", err)
	}

	return toProto(updatedCart), nil
}

func toProto(cart domain.Cart) *cartv1.Cart {
	items := make([]*cartv1.CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		items = append(items, &cartv1.CartItem{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	return &cartv1.Cart{
		Id:            cart.ID,
		UserId:        cart.UserID,
		Status:        cart.Status,
		Items:         items,
		CreatedAtUnix: cart.CreatedAt.Unix(),
		UpdatedAtUnix: cart.UpdatedAt.Unix(),
	}
}
