package rpc


import (
	"context"
	"time"
	"fmt"

	"dimi/kkalcs/mlapi/orders"
	pb "dimi/kkalcs/pb/orderspb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedOrderServiceServer
}

func (s *Server) GetTotalOrders(ctx context.Context, req *pb.OrderRequest) (*pb.OrderResponse, error) {
	if req.GetYear1() == 0 || req.GetMonth1() == 0 || req.GetYear2() == 0 || req.GetMonth2() == 0 {
		return nil, status.Error(codes.InvalidArgument, "Missing date parameters")
	}
	if req.GetMonth1() < 1 || req.GetMonth1() > 12 || req.GetMonth2() < 1 || req.GetMonth2() > 12 {
		return nil, status.Error(codes.InvalidArgument, "Invalid month parameter")
	}
	if req.GetYear1() > req.GetYear2() || (req.GetYear1() == req.GetYear2() && req.GetMonth1() > req.GetMonth2()) {
		return nil, status.Error(codes.InvalidArgument, "Invalid date range")
	}

	dateFrom := time.Date(int(req.Year1), time.Month(req.Month1), 21, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(int(req.Year2), time.Month(req.Month2), 22, 0, 0, 0, 0, time.UTC).Add(-1 * time.Nanosecond)

	data, err := orders.FetchAll(dateFrom, dateTo)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch orders: %v", err)
	}
	typedResultAny := orders.Total(data)

	typedResult, ok := typedResultAny.(struct { TotalBruto float64; SaleFeeTotal float64; MedianTax float64; TotalLiquido float64 })
	if !ok {
		return nil, status.Error(codes.Internal, "failed to cast result from orders.Total")
	}

	return &pb.OrderResponse{
		TotalBruto:      typedResult.TotalBruto,
		SaleFeeTotal:    typedResult.SaleFeeTotal,
		MedianTax:       typedResult.MedianTax,
		TotalLiquido:    typedResult.TotalLiquido,
		PeriodProcessed: fmt.Sprintf("%d/%d a %d/%d", req.Month1, req.Year1, req.Month2, req.Year2),
	}, nil
}