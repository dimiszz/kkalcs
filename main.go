package main

import (
	"fmt"
	"log/slog"
	"time"

	"dimi/kkalcs/api"
	"dimi/kkalcs/dotenv"
	"dimi/kkalcs/logger"
	"dimi/kkalcs/mlapi/auth"
	"dimi/kkalcs/mlapi/orders"
	"dimi/kkalcs/mlapi/requests"
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

	err := api.Run()
	if err != nil {
		slog.Error("Error in code execution", "error", err)
	}
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
