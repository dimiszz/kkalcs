package api

import (
	"dimi/kkalcs/logger"
	"dimi/kkalcs/mlapi/orders"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

func Run() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/orders", getOrders)

	server := &http.Server{
		Addr:    ":8080",
		Handler: loggerMdwr(mux),
	}
	slog.Info("Starting server on :8080")
	err := server.ListenAndServe()
	return err
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	// Handle the request to get orders here
	// You can use the requests package to make API calls to Mercado Libre
	// and return the response to the client.

	data, err := orders.FetchAll()
	if err != nil {
		slog.Error("Failed to fetch orders", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("Failed to marshal orders", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func loggerMdwr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String() // You can generate a UUID instead
			r.Header.Set("X-Request-ID", requestID)
		}

		logger.SetRequestID(requestID)
		next.ServeHTTP(w, r)
		logger.ResetRequestID()
	})
}
