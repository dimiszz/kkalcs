package rpc


import (
	"context"
	"time"
	"encoding/json"
	"fmt"

	"dimi/kkalcs/mlapi/orders" // Seu pacote de negócio
	pb "dimi/kkalcs/pb/orderspb" // Pacote gerado pelo protoc

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type totalsPayload struct {
	TotalBruto   float64
	SaleFeeTotal float64
	MedianTax    float64
	TotalLiquido float64
}

type Server struct {
	pb.UnimplementedOrderServiceServer
}

func (s *Server) GetTotalOrders(ctx context.Context, req *pb.OrderRequest) (*pb.OrderResponse, error) {
	// 1. Toda a validação e lógica de negócio que estava no seu handler HTTP agora vive aqui.
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

	// 2. Chamadas para o seu pacote de negócio
	data, err := orders.FetchAll(dateFrom, dateTo)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch orders: %v", err)
	}
	// 3. Chamamos sua função original, que retorna `any`
	anonymousResult := orders.Total(data)

	// 4. Convertemos o resultado `any` para um array de bytes JSON.
	jsonBytes, err := json.Marshal(anonymousResult)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal anonymous result: %v", err)
	}

	// 5. Lemos os bytes JSON para a nossa struct local `totalsPayload`.
	var typedResult totalsPayload
	if err := json.Unmarshal(jsonBytes, &typedResult); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal to typed result: %v", err)
	}

	// 6. Agora podemos usar `typedResult` com segurança!
	return &pb.OrderResponse{
		TotalBruto:      typedResult.TotalBruto,
		SaleFeeTotal:    typedResult.SaleFeeTotal,
		MedianTax:       typedResult.MedianTax,
		TotalLiquido:    typedResult.TotalLiquido,
		PeriodProcessed: fmt.Sprintf("%d/%d a %d/%d", req.Month1, req.Year1, req.Month2, req.Year2),
	}, nil
}