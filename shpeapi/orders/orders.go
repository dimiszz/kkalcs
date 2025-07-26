package orders

import (
	"encoding/json"
	"fmt"
	"io/ioutil" // Using ioutil for simplicity, consider os.ReadFile for Go 1.16+
	"strings"
	"time"

	"dimi/kkalcs/shpeapi/auth"     // Your existing auth package for Shopee
	"dimi/kkalcs/shpeapi/requests" // The adapted requests package
)

const shopeeBaseURL = "https://partner.shopeemobile.com"

// ShopeeOrderItem represents an item within a Shopee order
type ShopeeOrderItem struct {
	ItemID               int64   `json:"item_id"`
	ItemName             string  `json:"item_name"`
	ModelID              int64   `json:"model_id"`
	ModelName            string  `json:"model_name"`
	ModelQuantity        int     `json:"model_quantity"`
	ModelOriginalPrice   float64 `json:"model_original_price"`
	ModelDiscountedPrice float64 `json:"model_discounted_price"`
	ActualShippingFee    float64 `json:"actual_shipping_fee"`
	BuyerPaidShippingFee float64 `json:"buyer_paid_shipping_fee"`
	CommissionFee        float64 `json:"commission_fee"`
	ServiceFee           float64 `json:"service_fee"`
	SellerCoinCashBack   float64 `json:"seller_coin_cash_back"`
}

// ShopeeOrder represents a Shopee order
type ShopeeOrder struct {
	OrderSN              string  `json:"order_sn"`
	OrderStatus          string  `json:"order_status"`
	CreateTime           int64   `json:"create_time"`
	UpdateTime           int64   `json:"update_time"`
	PayTime              int64   `json:"pay_time"`
	BuyerUserID          int64   `json:"buyer_user_id"`
	BuyerUsername        string  `json:"buyer_username"`
	TotalAmount          float64 `json:"total_amount"`
	Currency             string  `json:"currency"`
	ShippingID           int64   `json:"shipping_id"`
	Region               string  `json:"region"`
	ActualShippingCost   float64 `json:"actual_shipping_cost"`
	BuyerPaidShippingFee float64 `json:"buyer_paid_shipping_fee"`
	SellerCoinCashBack   float64 `json:"seller_coin_cash_back"`
	CommissionFee        float64 `json:"commission_fee"`
	ServiceFee           float64 `json:"service_fee"`
	PaymentInfo          struct {
		PayableAmount float64 `json:"payable_amount"`
	} `json:"payment_info"`
	ShopeeOrderItems []ShopeeOrderItem `json:"items"`
}

// FetchAllShopeeOrders fetches all Shopee orders within a given date range.
// It batches calls to GetShopeeOrderDetail to reduce the total number of API requests.
func FetchAllShopeeOrders(dateFrom, dateTo time.Time) ([]ShopeeOrder, error) {
	const maxDateRangeDays = 15 // Shopee's API limit for time_from and time_to
	const pageSize = 100        // Maximum page size for get_order_list
	const batchSize = 50        // Number of order SNs to fetch details for in one GetShopeeOrderDetail call

	allOrders := []ShopeeOrder{}
	shopID := auth.GetUserID()

	currentDate := dateFrom
	for currentDate.Before(dateTo) || currentDate.Equal(dateTo) {
		endOfPeriod := currentDate.Add(maxDateRangeDays * 24 * time.Hour)
		if endOfPeriod.After(dateTo) {
			endOfPeriod = dateTo
		}

		fmt.Printf("Fetching orders from %s to %s\n", currentDate.Format("2006-01-02"), endOfPeriod.Format("2006-01-02"))

		cursor := ""
		for {
			orderSNs, nextCursor, err := FetchShopeeOrderList(currentDate, endOfPeriod, pageSize, cursor, shopID)
			if err != nil {
				return nil, fmt.Errorf("error fetching Shopee order list page: %w", err)
			}

			// Batch fetch details for multiple orders
			for i := 0; i < len(orderSNs); i += batchSize {
				end := i + batchSize
				if end > len(orderSNs) {
					end = len(orderSNs)
				}
				batchSNs := orderSNs[i:end]

				detailedOrders, err := GetShopeeOrderDetailsBatch(batchSNs)
				if err != nil {
					fmt.Printf("Warning: Could not fetch details for batch of orders %v: %v. Continuing with next batch.\n", batchSNs, err)
					// Decide how to handle batch errors: skip batch, retry, log and continue.
					// For now, we log and continue to next batch.
					continue
				}
				allOrders = append(allOrders, detailedOrders...)
				time.Sleep(50 * time.Millisecond) // Small delay between batch detail fetches
			}

			if nextCursor == "" {
				break
			}
			cursor = nextCursor
			time.Sleep(100 * time.Millisecond) // Be nice to the API between pages of order lists
		}
		currentDate = endOfPeriod.Add(24 * time.Hour)
	}

	err := saveShopeeOrdersToFile(allOrders, "all_shopee_orders.json")
	if err != nil {
		fmt.Printf("Warning: Could not save Shopee orders to file: %v\n", err)
	}

	return allOrders, nil
}

// FetchShopeeOrderList makes a single request to Shopee's get_order_list API
// and returns only the order SNs (IDs) to minimize data fetched initially.
func FetchShopeeOrderList(dateFrom, dateTo time.Time, pageSize int, cursor string, shopID string) ([]string, string, error) {
	apiPath := "/api/v2/order/get_order_list"

	timeFromUnix := dateFrom.Unix()
	timeToUnix := dateTo.Unix()

	orderStatus := "COMPLETED"

	queryParams := requests.NewQueryParams()
	queryParams.Add("time_from", fmt.Sprintf("%d", timeFromUnix))
	queryParams.Add("time_to", fmt.Sprintf("%d", timeToUnix))
	queryParams.Add("page_size", fmt.Sprintf("%d", pageSize))
	queryParams.Add("order_status", orderStatus)
	queryParams.Add("time_range_field", "update_time")
	if cursor != "" {
		queryParams.Add("cursor", cursor)
	}

	body, err := requests.MakeShopeeRequest(requests.GET, shopeeBaseURL, apiPath, queryParams, nil)
	if err != nil {
		return nil, "", fmt.Errorf("error making Shopee API request: %w", err)
	}

	fmt.Println("Corpo da resposta:", string(body))

	var response struct {
		RequestID string `json:"request_id"`
		Error     string `json:"error"`
		Message   string `json:"message"`
		Response  struct {
			OrderList []struct {
				OrderSN string `json:"order_sn"`
			} `json:"order_list"`
			More       bool   `json:"more"`
			NextCursor string `json:"next_cursor"`
		} `json:"response"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, "", fmt.Errorf("error unmarshaling Shopee order list response: %w", err)
	}

	if response.Error != "" {
		return nil, "", fmt.Errorf("Shopee API error: %s - %s", response.Error, response.Message)
	}

	var orderSNs []string
	for _, rawOrder := range response.Response.OrderList {
		orderSNs = append(orderSNs, rawOrder.OrderSN)
	}

	return orderSNs, response.Response.NextCursor, nil
}

// GetShopeeOrderDetailsBatch fetches details for multiple Shopee orders in a single API call.
func GetShopeeOrderDetailsBatch(orderSNs []string) ([]ShopeeOrder, error) {
	if len(orderSNs) == 0 {
		return nil, nil // No orders to fetch
	}

	apiPath := "/api/v2/order/get_order_detail"

	// Join the order SNs into a comma-separated string
	orderSNList := strings.Join(orderSNs, ",")

	queryParams := requests.NewQueryParams()
	queryParams.Add("order_sn_list", orderSNList) // Corrected parameter name

	queryParams.Add("response_optional_fields", "item_list,actual_shipping_cost,buyer_paid_shipping_fee,seller_coin_cash_back,commission_fee,service_fee,payment_info")

	body, err := requests.MakeShopeeRequest(requests.GET, shopeeBaseURL, apiPath, queryParams, nil)
	if err != nil {
		return nil, fmt.Errorf("error making Shopee order detail API request for batch %v: %w", orderSNs, err)
	}

	var response struct {
		RequestID string `json:"request_id"`
		Error     string `json:"error"`
		Message   string `json:"message"`
		Response  struct {
			OrderList []struct { // get_order_detail returns an array of orders
				OrderSN              string            `json:"order_sn"`
				OrderStatus          string            `json:"order_status"`
				CreateTime           int64             `json:"create_time"`
				UpdateTime           int64             `json:"update_time"`
				PayTime              int64             `json:"pay_time"`
				BuyerUserID          int64             `json:"buyer_user_id"`
				BuyerUsername        string            `json:"buyer_username"`
				TotalAmount          float64           `json:"total_amount"`
				ActualShippingCost   float64           `json:"actual_shipping_cost"`
				BuyerPaidShippingFee float64           `json:"buyer_paid_shipping_fee"`
				SellerCoinCashBack   float64           `json:"seller_coin_cash_back"`
				CommissionFee        float64           `json:"commission_fee"`
				ServiceFee           float64           `json:"service_fee"`
				Currency             string            `json:"currency"`
				ShippingID           int64             `json:"shipping_id"`
				Region               string            `json:"region"`
				Items                []ShopeeOrderItem `json:"item_list"`
				PaymentInfo          struct {
					PayableAmount float64 `json:"payable_amount"`
				} `json:"payment_info"`
			} `json:"order_list"`
		} `json:"response"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling Shopee order detail batch response: %w", err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("Shopee API error for batch %v: %s - %s", orderSNs, response.Error, response.Message)
	}

	var orders []ShopeeOrder
	for _, rawOrder := range response.Response.OrderList {
		order := ShopeeOrder{
			OrderSN:              rawOrder.OrderSN,
			OrderStatus:          rawOrder.OrderStatus,
			CreateTime:           rawOrder.CreateTime,
			UpdateTime:           rawOrder.UpdateTime,
			PayTime:              rawOrder.PayTime,
			BuyerUserID:          rawOrder.BuyerUserID,
			BuyerUsername:        rawOrder.BuyerUsername,
			TotalAmount:          rawOrder.TotalAmount,
			Currency:             rawOrder.Currency,
			ShippingID:           rawOrder.ShippingID,
			Region:               rawOrder.Region,
			ActualShippingCost:   rawOrder.ActualShippingCost,
			BuyerPaidShippingFee: rawOrder.BuyerPaidShippingFee,
			SellerCoinCashBack:   rawOrder.SellerCoinCashBack,
			CommissionFee:        rawOrder.CommissionFee,
			ServiceFee:           rawOrder.ServiceFee,
			PaymentInfo:          rawOrder.PaymentInfo,
			ShopeeOrderItems:     rawOrder.Items,
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// CalculateShopeeOrderMetrics calculates total sales and estimated profit for Shopee orders.
func CalculateShopeeOrderMetrics(orders []ShopeeOrder) struct {
	TotalGrossSales    float64
	TotalFees          float64
	TotalShippingCosts float64
	TotalNetSales      float64
} {
	var totalGrossSales float64
	var totalFees float64
	var totalShippingCosts float64

	for _, order := range orders {
		totalGrossSales += order.TotalAmount
		totalFees += order.CommissionFee
		totalFees += order.ServiceFee
		totalShippingCosts += order.ActualShippingCost
	}

	totalNetSales := totalGrossSales - totalFees - totalShippingCosts

	return struct {
		TotalGrossSales    float64
		TotalFees          float64
		TotalShippingCosts float64
		TotalNetSales      float64
	}{
		TotalGrossSales:    totalGrossSales,
		TotalFees:          totalFees,
		TotalShippingCosts: totalShippingCosts,
		TotalNetSales:      totalNetSales,
	}
}

// saveShopeeOrdersToFile is a helper to save fetched orders to a JSON file.
func saveShopeeOrdersToFile(orders []ShopeeOrder, filename string) error {
	asJSON, err := json.MarshalIndent(orders, "", "\t")
	if err != nil {
		return fmt.Errorf("error marshaling Shopee orders to JSON: %w", err)
	}

	err = ioutil.WriteFile(filename, asJSON, 0644)
	if err != nil {
		return fmt.Errorf("error writing Shopee orders to file %s: %w", filename, err)
	}
	return nil
}
