package main

import (
	"encoding/json"
	"fmt"

	"dimi/kkalcs/dotenv"
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

	err := run()
	if err != nil {
		fmt.Println("Erro:", err)
	}
}

func run() error {

	//err := CalculateProfit()

	orders.Get("2000010876085454")

	return nil
}

func getAllItemsIds() ([]string, error) {
	const limit = 50
	var itemsId []string
	offset := 0

	for {
		url := fmt.Sprintf("https://api.mercadolibre.com/users/%s/items/search?offset=%d&limit=%d", requests.USER_ID, offset, limit)

		fmt.Println("URL:", url)

		body, err := requests.MakeSimpleRequest(requests.GET, url, nil)
		if err != nil {
			return nil, fmt.Errorf("erro ao fazer requisição: %s", err)
		}

		var result SearchResult
		err = json.Unmarshal(body, &result)
		if err != nil {
			return nil, fmt.Errorf("erro ao parsear JSON: %s", err)
		}

		itemsId = append(itemsId, result.Results...)

		if len(result.Results) < limit {
			break
		}
		offset += limit
	}

	return itemsId, nil
}

func GetRateLimit() error {
	urla := "https://api.mercadolibre.com/marketplace/users/cap"

	body, err := requests.MakeSimpleRequest(requests.GET, urla, nil)
	if err != nil {
		return fmt.Errorf("erro ao fazer requisição: %s", err)
	}
	fmt.Println("Corpo da resposta:", string(body))
	return nil
}

func LoadUserId() {
	access_token := auth.GetAcessToken()
	start := len(access_token) - 10
	userID := access_token[start:]
	requests.USER_ID = userID
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
	ords, err := orders.FetchAll()
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
