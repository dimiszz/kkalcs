package orders

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"dimi/kkalcs/mlapi/requests"
)

type OrderItem struct {
	ItemID        string  `json:"item_id"`
	CategoryID    string  `json:"category_id"`
	Quantity      int     `json:"quantity"`
	UnitPrice     float64 `json:"unit_price"`
	ListingTypeID string  `json:"listing_type_id"`
	SaleFee       float64 `json:"sale_fee"`
	SKU           string  `json:"seller_sku"`
}

type Order struct {
	OrderID     int64       `json:"order_id"`
	PaidAmount  float64     `json:"paid_amount"`
	Status      string      `json:"status"`
	DateCreated string      `json:"date_created"`
	ShippingID  int         `json:"shipping_id"`
	Items       []OrderItem `json:"items"`
}

func FetchAll() ([]Order, error) {
	const limit = 50
	offset := 0

	dateFrom := "2025-02-21T00:00:00Z"
	dateTo := "2025-03-21T23:59:59Z"

	// Status válidos para excluir pedidos cancelados
	statuses := []string{"paid", "confirmed"}
	validStatuses := strings.Join(statuses, ",")

	all_ords := []Order{}

	for {
		url := fmt.Sprintf(
			"https://api.mercadolibre.com/orders/search?seller=%s&limit=%d&offset=%d&order.date_created.from=%s&order.date_created.to=%s&order.status=%s",
			requests.USER_ID, limit, offset, dateFrom, dateTo, validStatuses,
		)

		//url := fmt.Sprintf("https://api.mercadolibre.com/orders/search?seller=%s&limit=%d&offset=%d&date_from=%s&date_to=%s", USER_ID, limit, offset, dateFrom, dateTo)
		fmt.Println("URL:", url)

		body, err := requests.MakeSimpleRequest(requests.GET, url, nil)

		if err != nil {
			return nil, fmt.Errorf("erro ao fazer requisição: %s", err)
		}

		ords, err := extract(body)
		if err != nil {
			return nil, fmt.Errorf("erro ao extrair pedidos: %s", err)
		}

		all_ords = append(all_ords, ords...)

		var paging struct {
			Total int `json:"total"`
		}
		err = json.Unmarshal(body, &paging)
		if err != nil {
			return nil, fmt.Errorf("erro ao parsear a resposta de paginação: %s", err)
		}

		// Se a quantidade de pedidos retornados for menor que o limite, significa que não há mais páginas
		if len(ords) < limit {
			break
		}

		// Incrementa o offset para pegar a próxima página
		offset += limit
		fmt.Println(offset)
	}

	f, err := os.Create("all_orderns.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	as_json, err := json.MarshalIndent(all_ords, "", "\t")
	if err != nil {
		return nil, err
	}

	f.Write(as_json)

	return all_ords, nil
}

func extract(data []byte) ([]Order, error) {
	var raw struct {
		Results []struct {
			ID           int64    `json:"id"`
			Status       string   `json:"status"`
			DateCreated  string   `json:"date_created"`
			ShippingCost *float64 `json:"shipping_cost"` // pode ser nulo
			PaidAmount   float64  `json:"paid_amount"`
			Shipping     struct {
				ID int `json:"id"`
			} `json:"shipping"`
			OrderItems []struct {
				Item struct {
					ID            string `json:"id"`
					CategoryID    string `json:"category_id"`
					ListingTypeID string `json:"listing_type_id"`
					SKU           string `json:"seller_sku"`
				} `json:"item"`
				Quantity  int     `json:"quantity"`
				UnitPrice float64 `json:"unit_price"`
				SaleFee   float64 `json:"sale_fee"`
			} `json:"order_items"`
		} `json:"results"`
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer unmarshal: %v", err)
	}

	var orders []Order
	for _, r := range raw.Results {
		order := Order{
			OrderID:     r.ID,
			Status:      r.Status,
			DateCreated: r.DateCreated,
			ShippingID:  r.Shipping.ID,
			PaidAmount:  r.PaidAmount,
		}

		for _, oi := range r.OrderItems {
			item := OrderItem{
				ItemID:        oi.Item.ID,
				CategoryID:    oi.Item.CategoryID,
				Quantity:      oi.Quantity,
				UnitPrice:     oi.UnitPrice,
				ListingTypeID: oi.Item.ListingTypeID,
				SaleFee:       oi.SaleFee,
				SKU:           oi.Item.SKU,
			}
			order.Items = append(order.Items, item)
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func Total(orders []Order) float64 {
	var total float64
	var sale_fee_total float64
	for _, order := range orders {
		for _, item := range order.Items {
			total += item.UnitPrice * float64(item.Quantity)
			sale_fee_total += item.SaleFee
		}
	}

	fmt.Println("Total bruto:", total)
	fmt.Println("Sale_fee_total: ", sale_fee_total)

	median_tax := sale_fee_total / total

	fmt.Println("Median Tax:", median_tax)

	fmt.Println("Total liquido:", total-sale_fee_total)

	return total - sale_fee_total
}

func Get(orderId string) {
	url := fmt.Sprintf("https://api.mercadolibre.com/orders/%s", orderId)

	body, err := requests.MakeSimpleRequest(requests.GET, url, nil)
	if err != nil {
		return
	}

	fmt.Println("BODY:", string(body))
}

func Fetch() ([]Order, error) {
	//url := fmt.Sprintf("https://api.mercadolibre.com/items?ids=%s&attributes=id,title,price,base_price,original_price", temp)
	//url := fmt.Sprintf("https://api.mercadolibre.com/items/%s/prices", itemsId[0])
	url := fmt.Sprintf("https://api.mercadolibre.com/orders/search?seller=%s", requests.USER_ID)
	fmt.Println("URL:", url)

	body, err := requests.MakeSimpleRequest(requests.GET, url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer requisição: %s", err)
	}

	ords, err := extract(body)
	if err != nil {
		return nil, fmt.Errorf("erro ao extrair pedidos: %s", err)
	}

	f, err := os.Create("fetch_orders.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	as_json, _ := json.MarshalIndent(ords, "", "\t")
	f.Write(as_json)

	Total(ords)

	fmt.Println("pedidos:", ords)

	return ords, nil
}
