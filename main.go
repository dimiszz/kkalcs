package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"dimi/kkalcs/dotenv"
	"dimi/kkalcs/mlapi/auth"
	"dimi/kkalcs/mlapi/categories"
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

var USER_ID string

func main() {
	dotenv.Load()
	LoadUserId()

	err := run()
	if err != nil {
		fmt.Println("Erro:", err)
	}

	auth.RegisterAuthResponse()
}

func run() error {

	// _, err := test.CreateTestUser()
	// if err != nil {
	// 	return fmt.Errorf("erro ao criar usuário de teste: %s", err)
	// }
	// Test()
	// return nil

	itemsId, err := getAllItemsIds()
	if err != nil {
		return fmt.Errorf("erro ao conseguir items: %s", err)
	}
	//fmt.Println("Items ID:", itemsId)
	//fmt.Println("Total de itens:", len(itemsId))

	err = getItemsDetails(itemsId)
	if err != nil {
		return fmt.Errorf("erro ao conseguir items: %s", err)
	}

	fmt.Println()

	cate, err := categories.GetListingPrices("MLB437616")
	if err != nil {
		return fmt.Errorf("erro ao conseguir items: %s", err)
	}
	fmt.Println("Prices: ", cate)

	return nil
}

func getAllItemsIds() ([]string, error) {
	const limit = 50
	var itemsId []string
	offset := 0

	for {
		url := fmt.Sprintf("https://api.mercadolibre.com/users/%s/items/search?offset=%d&limit=%d", USER_ID, offset, limit)

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

func getItemsDetails(itemsId []string) error {

	temp := strings.Join(itemsId, ",")
	fmt.Println(temp)
	// =$ITEM_ID1,$ITEM_ID2

	//url := fmt.Sprintf("https://api.mercadolibre.com/items?ids=%s&attributes=id,title,price,base_price,original_price", temp)
	//url := fmt.Sprintf("https://api.mercadolibre.com/items/%s/prices", itemsId[0])
	url := fmt.Sprintf("https://api.mercadolibre.com/orders/search?seller=%s", USER_ID)
	fmt.Println("URL:", url)

	body, err := requests.MakeSimpleRequest(requests.GET, url, nil)
	if err != nil {
		return fmt.Errorf("erro ao fazer requisição: %s", err)
	}

	ords, err := orders.ExtractOrders(body)
	if err != nil {
		return fmt.Errorf("erro ao extrair pedidos: %s", err)
	}

	fmt.Println("pedidos:", ords)

	return nil
}

func LoadUserId() {
	access_token := auth.GetAcessToken()
	start := len(access_token) - 9
	userID := access_token[start:]
	USER_ID = userID
}

func Test() {
	urla := "https://api.mercadolibre.com/orders/reports"

	fromDate := "2023-01-01T00:00:00.000Z"
	toDate := "2024-12-31T23:59:59.999Z"

	data := url.Values{}
	data.Set("filters.date_created.from", fromDate)
	data.Set("filters.date_created.to", toDate)
	data.Set("fields", "id,date_created,total_amount,status,order_items,shipping_cost,tags")

	body, err := requests.MakeSimpleRequest(requests.GET, urla, data)
	if err != nil {
		fmt.Println("Erro ao fazer requisição:", err)
		return
	}
	fmt.Println("Corpo da resposta:", string(body))
}
