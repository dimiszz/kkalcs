package api

import (
	"dimi/kkalcs/logger"
	"dimi/kkalcs/mlapi/orders"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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
	query := r.URL.Query()

	year1Str := query.Get("year1")
	month1Str := query.Get("month1")
	year2Str := query.Get("year2")
	month2Str := query.Get("month2")

	if year1Str == "" || month1Str == "" || year2Str == "" || month2Str == "" {
		http.Error(w, "Missing date parameters. Required: year1, month1, year2, month2", http.StatusBadRequest)
		return
	}
	year1, err := strconv.Atoi(year1Str)
	if err != nil {
		http.Error(w, "Invalid year1 parameter", http.StatusBadRequest)
		return
	}
	month1, err := strconv.Atoi(month1Str)
	if err != nil || month1 < 1 || month1 > 12 {
		http.Error(w, "Invalid month1 parameter", http.StatusBadRequest)
		return
	}
	year2, err := strconv.Atoi(year2Str)
	if err != nil {
		http.Error(w, "Invalid year2 parameter", http.StatusBadRequest)
		return
	}
	month2, err := strconv.Atoi(month2Str)
	if err != nil || month2 < 1 || month2 > 12 {
		http.Error(w, "Invalid month2 parameter", http.StatusBadRequest)
		return
	}
	if year1 > year2 || (year1 == year2 && month1 > month2) {
		http.Error(w, "Invalid date range", http.StatusBadRequest)
		return
	}

	dateFrom := time.Date(year1, time.Month(month1), 21, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(year2, time.Month(month2), 22, 0, 0, 0, 0, time.UTC).Add(-1 * time.Nanosecond)
	slog.Info("Fetching orders", "dateFrom", dateFrom, "dateTo", dateTo)

	data, err := orders.FetchAll(dateFrom, dateTo)
	if err != nil {
		slog.Error("Failed to fetch orders", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	result := orders.Total(data)
	jsonResult, err := json.Marshal(result)
	if err != nil {
		slog.Error("Failed to marshal orders", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(jsonResult))
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
