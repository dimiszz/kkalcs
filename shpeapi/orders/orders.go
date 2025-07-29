package orders

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"dimi/kkalcs/shpeapi/requests"
)

const (
	shopeeBaseURL        = "https://partner.shopeemobile.com"
	orderListAPIPath     = "/api/v2/order/get_order_list"
	orderDetailAPIPath   = "/api/v2/order/get_order_detail"
	orderDetailBatchSize = 50  // Máximo de order_sn por chamada na API de detalhes
	orderListBatchSize   = 100 // Máximo de pedidos por página na API de lista
)

// ... (structs ShopeeOrderItem e ShopeeOrder permanecem as mesmas) ...
type ShopeeOrderItem struct {
	ItemName             string  `json:"item_name"`
	ModelName            string  `json:"model_name"`
	ModelSKU             string  `json:"model_sku"`
	ModelQuantity        int     `json:"model_quantity_purchased"`
	ModelOriginalPrice   float64 `json:"model_original_price"`
	ModelDiscountedPrice float64 `json:"model_discounted_price"`
}

type ShopeeOrder struct {
	OrderSN          string            `json:"order_sn"`
	OrderStatus      string            `json:"order_status"`
	TotalAmount      float64           `json:"total_amount"`
	CreateTime       int64             `json:"create_time"`
	ShopeeOrderItems []ShopeeOrderItem `json:"item_list"`
}

// GetOrderListByDateRange busca a lista de order_sn com base em um período.
// Esta função é o novo ponto de entrada para a reconciliação.
func GetOrderListByDateRange(dateFrom, dateTo time.Time) ([]string, error) {
	var allOrderSNs []string
	cursor := "" // O cursor é usado para paginar através dos resultados

	slog.Info("Fetching order list by date range", "from", dateFrom, "to", dateTo)

	for {
		queryParams := requests.NewQueryParams()
		queryParams.Add("time_range_field", "create_time")
		queryParams.Add("time_from", fmt.Sprintf("%d", dateFrom.Unix()))
		queryParams.Add("time_to", fmt.Sprintf("%d", dateTo.Unix()))
		queryParams.Add("page_size", fmt.Sprintf("%d", orderListBatchSize))
		queryParams.Add("cursor", cursor)
		// Opcional: filtrar por status de pedido, ex: "COMPLETED"
		// queryParams.Add("order_status", "COMPLETED")

		body, err := requests.MakeShopeeRequest(requests.GET, shopeeBaseURL, orderListAPIPath, queryParams, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch order list page: %w", err)
		}

		var response struct {
			Error    string `json:"error"`
			Message  string `json:"message"`
			Response struct {
				More       bool   `json:"more"`
				NextCursor string `json:"next_cursor"`
				OrderList  []struct {
					OrderSN string `json:"order_sn"`
				} `json:"order_list"`
			} `json:"response"`
		}

		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal order list response: %w", err)
		}

		if response.Error != "" {
			return nil, fmt.Errorf("shopee API error fetching order list: %s - %s", response.Error, response.Message)
		}

		for _, order := range response.Response.OrderList {
			allOrderSNs = append(allOrderSNs, order.OrderSN)
		}

		if !response.Response.More {
			break // Sai do loop se não houver mais páginas
		}
		cursor = response.Response.NextCursor
	}

	slog.Info("Successfully fetched all order SNs for the period.", "count", len(allOrderSNs))
	return allOrderSNs, nil
}

// FetchOrderDetailsBySN busca os detalhes de uma lista específica de order_sn.
// Esta é a função que o pacote de reconciliação irá chamar.
func FetchOrderDetailsBySN(orderSNs []string) (map[string]ShopeeOrder, error) {
	orderMap := make(map[string]ShopeeOrder)
	if len(orderSNs) == 0 {
		return orderMap, nil
	}

	slog.Info("Fetching order details in batches by SN list...")
	for i := 0; i < len(orderSNs); i += orderDetailBatchSize {
		end := i + orderDetailBatchSize
		if end > len(orderSNs) {
			end = len(orderSNs)
		}
		batchSNs := orderSNs[i:end]

		slog.Info("Processing detail batch", "start_index", i, "size", len(batchSNs))

		queryParams := requests.NewQueryParams()
		queryParams.Add("order_sn_list", strings.Join(batchSNs, ","))
		// Buscamos apenas os campos que o pacote 'payments' não nos fornece.
		queryParams.Add("response_optional_fields", "item_list,total_amount,order_status,create_time")

		body, err := requests.MakeShopeeRequest(requests.GET, shopeeBaseURL, orderDetailAPIPath, queryParams, nil)
		if err != nil {
			slog.Error("Request to get_order_detail failed for batch, skipping.", "batch_sns", batchSNs, "error", err)
			continue // Pula para o próximo lote em caso de erro
		}

		var response struct {
			Response struct {
				OrderList []ShopeeOrder `json:"order_list"`
			} `json:"response"`
			Error   string `json:"error"`
			Message string `json:"message"`
		}

		if err := json.Unmarshal(body, &response); err != nil {
			slog.Error("Failed to unmarshal get_order_detail batch response, skipping.", "error", err)
			continue
		}

		if response.Error != "" {
			slog.Error("Shopee API error in get_order_detail, skipping batch.", "error", response.Error, "message", response.Message)
			continue
		}

		// Adiciona os pedidos do lote ao mapa.
		for _, order := range response.Response.OrderList {
			orderMap[order.OrderSN] = order
		}

		time.Sleep(100 * time.Millisecond)
	}

	slog.Info("Finished fetching order details.", "total_details_fetched", len(orderMap))
	return orderMap, nil
}
