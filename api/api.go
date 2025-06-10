package api

import (
	"dimi/kkalcs/logger"
	grpc_orders "dimi/kkalcs/rpc"
	pb "dimi/kkalcs/pb/orderspb"

	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var grpcClient pb.OrderServiceClient

func InitGRPCClient() error {
	conn, err := grpc.NewClient(":50050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	grpcClient = pb.NewOrderServiceClient(conn)
	return nil
}

func getOrdersFromGRPC(r *http.Request) (*pb.OrderResponse, error) {
	year1, err1 := strconv.Atoi(r.FormValue("year1"))
	month1, err2 := strconv.Atoi(r.FormValue("month1"))
	year2, err3 := strconv.Atoi(r.FormValue("year2"))
	month2, err4 := strconv.Atoi(r.FormValue("month2"))
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		slog.Error("invalid or missing date parameters", "err1", err1, "err2", err2, "err3", err3, "err4", err4)
		return nil, fmt.Errorf("invalid or missing date parameters")
	}

	req := &pb.OrderRequest{
		Year1:  int32(year1),
		Month1: int32(month1),
		Year2:  int32(year2),
		Month2: int32(month2),
	}

	return grpcClient.GetTotalOrders(r.Context(), req)
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	resp, err := getOrdersFromGRPC(r)
	if err != nil {
		slog.Error("gRPC call failed", "error", err)
		http.Error(w, "Internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func getOrdersHtmxView(w http.ResponseWriter, r *http.Request) {
	resp, err := getOrdersFromGRPC(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<p class="error">Erro ao buscar dados: ` + err.Error() + `</p>`))
		return
	}

	tmpl, _ := template.New("orderResult").Parse(`
		<div id="result-card">
			<h4>Resultados para o Período: {{.PeriodProcessed}}</h4>
			<p><strong>Total Bruto:</strong> R$ {{printf "%.2f" .TotalBruto}}</p>
			<p><strong>Total Taxas:</strong> R$ {{printf "%.2f" .SaleFeeTotal}}</p>
			<p><strong>Taxa Média:</strong> {{printf "%.2f" .MedianTax}}%</p>
			<hr>
			<p><strong>Total Líquido:</strong> R$ {{printf "%.2f" .TotalLiquido}}</p>
		</div>
	`)
	tmpl.Execute(w, resp)
}

func serveIndexPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func Run() error {
	go func() {
		lis, err := net.Listen("tcp", ":50050")
		if err != nil {
			log.Fatalf("failed to listen on gRPC port: %v", err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterOrderServiceServer(grpcServer, &grpc_orders.Server{})
		slog.Info("Starting gRPC server on :50050")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()
	err := InitGRPCClient()

	if err != nil {
		slog.Error("Failed to initialize gRPC client", "error", err)
		return err
	}


	mux := http.NewServeMux()
	mux.HandleFunc("GET /", serveIndexPage)
	mux.HandleFunc("GET /api/v1/orders", getOrders)
	mux.HandleFunc("POST /api/v1/orders-view", getOrdersHtmxView)

	server := &http.Server{
		Addr:    ":8080",
		Handler: loggerMdwr(mux),
	}
	slog.Info("Starting HTTP server on :8080")
	return server.ListenAndServe()
}

func loggerMdwr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			r.Header.Set("X-Request-ID", requestID)
		}
		logger.SetRequestID(requestID)
		next.ServeHTTP(w, r)
		logger.ResetRequestID()
	})
}
