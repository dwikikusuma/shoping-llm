package grpc

import (
	"context"

	orderv1 "github.com/dwikikusuma/shoping-llm/api/gen/order/v1"
	"github.com/dwikikusuma/shoping-llm/internal/order/app"
	"github.com/dwikikusuma/shoping-llm/internal/order/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	orderv1.UnimplementedOrderServiceServer
	svc *app.Service
}

func NewServer(svc *app.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	if req.Items == nil || len(req.Items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "items must not be empty")
	}

	orderRequest := s.mapProtoToCreateOrderReq(req)
	order, err := s.svc.CreateOrder(ctx, orderRequest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
	}
	return &orderv1.CreateOrderResponse{
		OrderId:       order.ID,
		Status:        order.Status,
		TotalAmount:   order.TotalAmount,
		CreatedAtUnix: order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *Server) mapProtoToCreateOrderReq(req *orderv1.CreateOrderRequest) domain.CreateOrderRequest {
	orderItems := make([]domain.OrderItemRequest, 0, len(req.Items))

	for _, item := range req.Items {
		orderItems = append(orderItems, domain.OrderItemRequest{
			ProductID:  item.ProductId,
			Name:       item.Name,
			UnitAmount: item.UnitAmount,
			Quantity:   item.Quantity,
		})
	}

	return domain.CreateOrderRequest{
		UserID:         req.UserId,
		Currency:       req.Currency,
		ShippingAmount: req.ShippingFee,
		Items:          orderItems,
	}
}
