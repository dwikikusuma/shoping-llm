package grpc

import (
	"context"
	"errors"

	checkoutv1 "github.com/dwikikusuma/shoping-llm/api/gen/checkout/v1"
	"github.com/dwikikusuma/shoping-llm/internal/checkout/app"
	"github.com/dwikikusuma/shoping-llm/internal/checkout/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	checkoutv1.UnimplementedCheckoutServiceServer
	svc *app.Service
}

func NewServer(svc *app.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) Quote(ctx context.Context, req *checkoutv1.QuoteRequest) (*checkoutv1.QuoteResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	q, err := s.svc.Quote(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, app.ErrEmptyCart) {
			return nil, status.Error(codes.NotFound, "cart is empty")
		}
		return nil, status.Errorf(codes.Internal, "quote failed: %v", err)
	}

	return toProto(q), nil
}

func toProto(q domain.Quote) *checkoutv1.QuoteResponse {
	lines := make([]*checkoutv1.QuoteLine, 0, len(q.Lines))
	for _, ln := range q.Lines {
		lines = append(lines, &checkoutv1.QuoteLine{
			ProductId: ln.ProductID,
			Name:      ln.Name,
			Quantity:  int32(ln.Quantity),
			UnitPrice: &checkoutv1.Money{Currency: ln.UnitPrice.Currency, Amount: ln.UnitPrice.Amount},
			LineTotal: &checkoutv1.Money{Currency: ln.LineTotal.Currency, Amount: ln.LineTotal.Amount},
		})
	}

	return &checkoutv1.QuoteResponse{
		Lines: lines,
		Total: &checkoutv1.Money{Currency: q.Total.Currency, Amount: q.Total.Amount},
	}
}
