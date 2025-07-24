package orders

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dimi/kkalcs/dotenv"
	"dimi/kkalcs/shpeapi/auth"
)

// Order defines the structure for a single order in the list.
type Order struct {
	OrderSN       string `json:"order_sn"`
	OrderStatus   string `json:"order_status"`
	UpdateTime    int64  `json:"update_time"`
	CreateTime    int64  `json:"create_time"`
	TotalAmount   string `json:"total_amount"`
	BuyerUsername string `json:"buyer_username"`
}

// OrderList is the nested object inside the main response.
type OrderList struct {
	More       bool    `json:"more"`
	NextCursor string  `json:"next_cursor"`
	OrderList  []Order `json:"order_list"`
}

// GetOrderListResponse is the top-level structure for the API response.
type GetOrderListResponse struct {
	Response  OrderList `json:"response"`
	RequestID string    `json:"request_id"`
	Error     string    `json:"error"`
	Message   string    `json:"message"`
}

// GetOrderList fetches recent orders from the last 3 days.
// GetOrderList fetches recent orders from the last 3 days.
func GetOrderList() (*GetOrderListResponse, error) {
	// 1. Get required credentials from your auth module
	accessToken := auth.GetAcessToken()
	shopIDStr := auth.GetUserID()
	partnerID := dotenv.Get("APP_ID_SHP")
	partnerKey := dotenv.Get("APP_SECRET_KEY_SHP")

	// 2. Prepare request parameters
	timestamp := time.Now().Unix()
	path := "/api/v2/order/get_order_list"

	// 3. Create the specific signature base string for this endpoint
	baseString := fmt.Sprintf("%s%s%d%s%s", partnerID, path, timestamp, accessToken, shopIDStr)
	sign := auth.CalculateHmacSha256(baseString, partnerKey)

	// 4. Build the URL with all query parameters
	baseURL := "https://partner.shopeemobile.com"
	fullURL := fmt.Sprintf("%s%s", baseURL, path)

	params := url.Values{}

	// FIX: Add partner_id back into the query string.
	params.Set("partner_id", partnerID)

	params.Set("timestamp", fmt.Sprintf("%d", timestamp))
	params.Set("access_token", accessToken)
	params.Set("shop_id", shopIDStr)
	params.Set("sign", sign)

	// Set order-specific filters (e.g., orders from the last 3 days)
	timeTo := time.Now().Unix()
	timeFrom := time.Now().AddDate(0, 0, -3).Unix()
	params.Set("time_range_field", "create_time")
	params.Set("time_from", fmt.Sprintf("%d", timeFrom))
	params.Set("time_to", fmt.Sprintf("%d", timeTo))
	params.Set("page_size", "10")

	// 5. Make the GET request
	req, err := http.NewRequest("GET", fullURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("Response Body:", string(respBody))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}

	// 6. Unmarshal the response
	var orderResponse GetOrderListResponse
	if err := json.Unmarshal(respBody, &orderResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if orderResponse.Error != "" {
		return nil, fmt.Errorf("API returned an error: %s - %s", orderResponse.Error, orderResponse.Message)
	}

	return &orderResponse, nil
}

// Item represents a single item within an order.
type Item struct {
	ItemName string `json:"item_name"`
	ItemSku  string `json:"item_sku"`
	ModelSku string `json:"model_sku"`
	Quantity int    `json:"model_quantity_purchased"`
}

// RecipientAddress contains the buyer's shipping address.
type RecipientAddress struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Town    string `json:"town"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zipcode"`
	Address string `json:"full_address"`
}

// OrderDetail contains the full information for a single order.
type OrderDetail struct {
	OrderSN          string           `json:"order_sn"`
	OrderStatus      string           `json:"order_status"`
	MessageToSeller  string           `json:"message_to_seller"`
	ItemList         []Item           `json:"item_list"`
	RecipientAddress RecipientAddress `json:"recipient_address"`
	PaymentMethod    string           `json:"payment_method"`
	TotalAmount      string           `json:"total_amount"`
}

// GetOrderDetailResponse is the top-level structure for the detail API response.
type GetOrderDetailResponse struct {
	Response struct {
		OrderList []OrderDetail `json:"order_list"`
	} `json:"response"`
	RequestID string `json:"request_id"`
	Error     string `json:"error"`
	Message   string `json:"message"`
}

// GetOrderDetail fetches detailed information for a list of order serial numbers.
func GetOrderDetail(orderSNs []string) (*GetOrderDetailResponse, error) {
	// 1. Get required credentials from your auth module
	accessToken := auth.GetAcessToken()
	shopIDStr := auth.GetUserID()
	partnerID := dotenv.Get("APP_ID_SHP")
	partnerKey := dotenv.Get("APP_SECRET_KEY_SHP")

	// 2. Prepare request parameters
	timestamp := time.Now().Unix()
	path := "/api/v2/order/get_order_detail"

	// 3. Create the specific signature base string for this endpoint
	baseString := fmt.Sprintf("%s%s%d%s%s", partnerID, path, timestamp, accessToken, shopIDStr)
	sign := auth.CalculateHmacSha256(baseString, partnerKey)

	// 4. Build the URL with all query parameters
	baseURL := "https://partner.shopeemobile.com"
	fullURL := fmt.Sprintf("%s%s", baseURL, path)

	params := url.Values{}
	params.Set("partner_id", partnerID)
	params.Set("timestamp", fmt.Sprintf("%d", timestamp))
	params.Set("access_token", accessToken)
	params.Set("shop_id", shopIDStr)
	params.Set("sign", sign)

	// Join the slice of order SNs into a single comma-separated string.
	params.Set("order_sn_list", strings.Join(orderSNs, ","))

	// Specify which optional fields you want the API to return.
	optionalFields := "item_list,recipient_address,payment_method"
	params.Set("response_optional_fields", optionalFields)

	// 5. Make the GET request
	req, err := http.NewRequest("GET", fullURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}

	// 6. Unmarshal the response
	var detailResponse GetOrderDetailResponse
	if err := json.Unmarshal(respBody, &detailResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if detailResponse.Error != "" {
		return nil, fmt.Errorf("API returned an error: %s - %s", detailResponse.Error, detailResponse.Message)
	}

	return &detailResponse, nil
}

func Chance() {
	// --- Authentication (No changes here) ---
	log.Println("Authenticating with Shopee...")
	_ = auth.GetAcessToken()
	log.Println("Authentication successful.")

	// --- 1. Get the list of Order SNs ---
	log.Println("Fetching order list...")
	orderListResponse, err := GetOrderList()
	if err != nil {
		log.Fatalf("Failed to get order list: %v", err)
	}

	if len(orderListResponse.Response.OrderList) == 0 {
		log.Println("No recent orders found.")
		return
	}

	// --- 2. Prepare the list of SNs for the detail call ---
	var orderSNs []string
	fmt.Println("--- Found Order SNs ---")
	for _, order := range orderListResponse.Response.OrderList {
		fmt.Println(order.OrderSN)
		orderSNs = append(orderSNs, order.OrderSN)
	}
	fmt.Println("-----------------------")

	// --- 3. Get the details for the collected SNs ---
	log.Println("Fetching order details...")
	detailResponse, err := GetOrderDetail(orderSNs)
	if err != nil {
		log.Fatalf("Failed to get order details: %v", err)
	}

	// --- 4. Print the detailed information ---
	fmt.Println("\n--- Order Details ---")
	for _, detail := range detailResponse.Response.OrderList {
		fmt.Printf("Order SN: %s\n", detail.OrderSN)
		fmt.Printf("  Status: %s\n", detail.OrderStatus)
		fmt.Printf("  Buyer: %s\n", detail.RecipientAddress.Name)
		fmt.Printf("  Items:\n")
		for _, item := range detail.ItemList {
			fmt.Printf("    - %s (Qty: %d)\n", item.ItemName, item.Quantity)
		}
		fmt.Println("---------------------")
	}
}
