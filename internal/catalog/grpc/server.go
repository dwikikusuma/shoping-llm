package grpc

import (
	"context"
	"errors"

	catalogv1 "github.com/dwikikusuma/shoping-llm/api/gen/catalog/v1"
	"github.com/dwikikusuma/shoping-llm/internal/catalog/app"
	"github.com/dwikikusuma/shoping-llm/internal/catalog/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	catalogv1.UnimplementedCatalogServiceServer
	svc *app.Service
}

func NewServer(svc *app.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) CreateProduct(ctx context.Context, req *catalogv1.CreateProductRequest) (*catalogv1.CreateProductResponse, error) {
	if req == nil || req.Price == nil {
		return nil, status.Error(codes.InvalidArgument, "missing body/price")
	}
	product, err := s.svc.CreateProduct(ctx, req.Name, req.Description, req.Price.Currency, req.Price.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}
	return &catalogv1.CreateProductResponse{
		Product: toProto(product),
	}, nil
}

func (s *Server) GetProduct(ctx context.Context, req *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error) {
	p, err := s.svc.GetProduct(ctx, req.GetId())
	if err != nil {
		return nil, mapErr(err)
	}
	return &catalogv1.GetProductResponse{Product: toProto(p)}, nil
}

func (s *Server) ListProducts(ctx context.Context, req *catalogv1.ListProductsRequest) (*catalogv1.ListProductsResponse, error) {
	products, next, err := s.svc.ListProducts(ctx, req.GetQuery(), int(req.GetLimit()), req.GetCursor())
	if err != nil {
		return nil, mapErr(err)
	}

	out := make([]*catalogv1.Product, 0, len(products))
	for _, p := range products {
		cp := p
		out = append(out, toProto(cp))
	}

	return &catalogv1.ListProductsResponse{Products: out, NextCursor: next}, nil
}

func toProto(p domain.Product) *catalogv1.Product {
	return &catalogv1.Product{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price: &catalogv1.Money{
			Currency: p.Price.Currency,
			Amount:   p.Price.Amount,
		},
		CreatedAtUnix: p.CreatedAt.Unix(),
		UpdatedAtUnix: p.UpdatedAt.Unix(),
	}
}

func mapErr(err error) error {
	if errors.Is(err, app.ErrInvalidInput) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, app.ErrNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	return status.Error(codes.Internal, "internal error")
}
