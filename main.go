package main

import (
	"fmt"
	"log/slog"
	"time"

	"dimi/kkalcs/dotenv"
	"dimi/kkalcs/logger"
	"dimi/kkalcs/mlapi/auth"
	"dimi/kkalcs/mlapi/orders"
	"dimi/kkalcs/mlapi/requests"
	shpauth "dimi/kkalcs/shpeapi/auth"
	shporder "dimi/kkalcs/shpeapi/orders"
)

type Paging struct {
	Total int `json:"total"`
}

type SearchResult struct {
	Paging  Paging   `json:"paging"`
	Results []string `json:"results"`
}

func main() {
	dotenv.Load()
	LoadUserId()
	setupLogger()
	fmt.Println(shpauth.GetAcessToken())
	simpleTest()
	// err := api.Run()
	// if err != nil {
	// 	slog.Error("Error in code execution", "error", err)
	// }
}

func run() error {

	err := CalculateProfit()

	//orders.Get("2000010876085454")

	// t := time.Now().Format("2006-01-02T00:00:00Z")
	// fmt.Println(t)
	// fmt.Println("2025-02-21T00:00:00Z")

	return err
}

func LoadUserId() {
	requests.USER_ID = auth.GetUserID()
}

func Test() {
	urla := "https://api.mercadolibre.com/orders/2000010821544300/discounts"

	body, err := requests.MakeSimpleRequest(requests.GET, urla, nil)
	if err != nil {
		fmt.Println("Erro ao fazer requisição:", err)
		return
	}
	fmt.Println("Corpo da resposta:", string(body))

	urla = "https://api.mercadolibre.com/orders/2000010821544300"
	body, err = requests.MakeSimpleRequest(requests.GET, urla, nil)
	if err != nil {
		fmt.Println("Erro ao fazer requisição:", err)
		return
	}
	fmt.Println("Corpo da resposta:", string(body))
}

func CalculateProfit() error {
	dateFrom := time.Date(2025, time.February, 21, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2025, time.March, 21, 23, 59, 59, 0, time.UTC)

	ords, err := orders.FetchAll(dateFrom, dateTo)
	if err != nil {
		return fmt.Errorf("erro ao buscar pedidos: %s", err)
	}

	// shipments_costs := make([]shipments.ShipmentCost, len(ords))
	// for _, ord := range ords {
	// 	if ord.Status == "cancelled" {
	// 		continue
	// 	}
	// 	if ord.ShippingID == 0 {
	// 		fmt.Println("Pedido sem ID de envio:", ord.OrderID)
	// 		continue
	// 	}
	// 	s, err := shipments.FetchCosts(strconv.Itoa(ord.ShippingID))
	// 	if err != nil {
	// 		fmt.Println(ord)
	// 		fmt.Println("Erro:", err, "SHIPMENT_ID: ", ord.ShippingID, " ORDER_ID: ", ord.OrderID)
	// 		continue
	// 	}
	// 	if s.FinalCost != 0 {
	// 		fmt.Println("Shipment: ", s, "Order ID:", ord.OrderID)
	// 		continue
	// 	}
	// 	shipments_costs = append(shipments_costs, *s)
	// }

	fmt.Println("Total de pedidos:", len(ords))
	orders.Total(ords)

	return nil
}

func setupLogger() {

	logger.SetupLogger()

	slog.Debug("isso é debug") // não será exibido
	slog.Info("isso é info")   // será exibido
	slog.Warn("isso é warn")   // será exibido
	slog.Error("isso é error") // será exibido
}

func temp() {
	fmt.Println("Starting Shopee Order Fetcher...")

	// --- 1. Initialize Authentication ---
	// The auth.GetAcessToken() function handles the entire authentication flow:
	// - Tries to load a saved token.
	// - If expired, refreshes it.
	// - If no token or refresh fails, it initiates the first-time browser-based authentication.
	fmt.Println("Initiating Shopee authentication flow...")
	accessToken := shpauth.GetAcessToken() // This will trigger browser login if needed
	if accessToken == "" {
		fmt.Println("Failed to obtain Shopee access token. Exiting.")
		return
	}
	fmt.Println("Successfully obtained Shopee access token.")

	// Get the authenticated Shop ID
	shopID := shpauth.GetUserID()
	fmt.Printf("Authenticated for Shopee Shop ID: %s\n", shopID)

	// --- 2. Define Date Range for Orders ---
	// Fetch orders for the last 30 days.
	// Adjust these dates as per your requirement.
	dateTo := time.Now().UTC()
	dateFrom := dateTo.AddDate(0, 0, -30) // Orders from 30 days ago to now

	fmt.Printf("\nFetching Shopee orders from %s to %s...\n",
		dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02"))

	// --- 3. Fetch All Orders ---
	// This function handles pagination and the 15-day date range limit internally.
	// It will also fetch detailed information for each order.
	shopeeOrders, err := shporder.FetchAllShopeeOrders(dateFrom, dateTo)
	if err != nil {
		fmt.Printf("Error fetching Shopee orders: %v\n", err)
		return
	}

	fmt.Printf("Successfully fetched %d Shopee orders.\n", len(shopeeOrders))

	// --- 4. Calculate Order Metrics ---
	metrics := shporder.CalculateShopeeOrderMetrics(shopeeOrders)

	fmt.Println("\n--- Shopee Order Metrics (Last 30 Days) ---")
	fmt.Printf("Total Gross Sales: %.2f\n", metrics.TotalGrossSales)
	fmt.Printf("Total Estimated Fees (Commission + Service): %.2f\n", metrics.TotalFees)
	fmt.Printf("Total Actual Shipping Costs (Paid by Seller): %.2f\n", metrics.TotalShippingCosts)
	fmt.Printf("Total Net Sales (Gross Sales - Fees - Shipping Costs): %.2f\n", metrics.TotalNetSales)

	// --- 5. Optional: Display a sample order's details ---
	if len(shopeeOrders) > 0 {
		fmt.Printf("\n--- Sample Order Details (%s) ---\n", shopeeOrders[0].OrderSN)
		sampleOrder := shopeeOrders[0] // Pick the first fetched order as a sample

		fmt.Printf("Order SN: %s\n", sampleOrder.OrderSN)
		fmt.Printf("Status: %s\n", sampleOrder.OrderStatus)
		fmt.Printf("Total Amount (Buyer Paid): %.2f %s\n", sampleOrder.TotalAmount, sampleOrder.Currency)
		fmt.Printf("Commission Fee: %.2f\n", sampleOrder.CommissionFee)
		fmt.Printf("Service Fee: %.2f\n", sampleOrder.ServiceFee)
		fmt.Printf("Actual Shipping Cost: %.2f\n", sampleOrder.ActualShippingCost)
		fmt.Printf("Buyer Paid Shipping Fee: %.2f\n", sampleOrder.BuyerPaidShippingFee)
		fmt.Printf("Number of Items: %d\n", len(sampleOrder.ShopeeOrderItems))

		if len(sampleOrder.ShopeeOrderItems) > 0 {
			fmt.Println("  Items:")
			for i, item := range sampleOrder.ShopeeOrderItems {
				fmt.Printf("    %d. %s - %s (Qty: %d, Price: %.2f)\n",
					i+1, item.ItemName, item.ModelName, item.ModelQuantity, item.ModelDiscountedPrice)
			}
		}
	} else {
		fmt.Println("\nNo Shopee orders found for the specified date range.")
	}

	fmt.Println("\nShopee Order Fetcher finished.")
}

func simpleTest() {

	shpauth.GetAcessToken()

	dateTo := time.Now().UTC()
	dateFrom := dateTo.AddDate(0, 0, -15) // Orders from 30 days ago to now
	shopId := shpauth.GetUserID()

	_, _, err := shporder.FetchShopeeOrderList(dateFrom, dateTo, 10, "", shopId)
	if err != nil {
		fmt.Printf("Error fetching Shopee order list: %v\n", err)
		return
	}

}
