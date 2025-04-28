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

	auth.RegisterAuthResponse()
}

func run() error {

	// _, err := test.CreateTestUser()
	// if err != nil {
	// 	return fmt.Errorf("erro ao criar usuário de teste: %s", err)
	//}
	//Test()
	//orders.FetchAll()

	err := orders.FetchAll()
	fmt.Println("Erro:", err)

	orders.FetchShipping("44636621027")

	return nil

	// itemsId, err := getAllItemsIds()
	// if err != nil {
	// 	return fmt.Errorf("erro ao conseguir items: %s", err)
	// }
	// //fmt.Println("Items ID:", itemsId)
	// //fmt.Println("Total de itens:", len(itemsId))

	// err = getItemsDetails(itemsId)
	// if err != nil {
	// 	return fmt.Errorf("erro ao conseguir items: %s", err)
	// }

	//GetOrders()

	fmt.Println()

	// cate, err := categories.GetListingPrices("MLB244658")
	// if err != nil {
	// 	return fmt.Errorf("erro ao conseguir items: %s", err)
	// }
	// fmt.Println("Prices: ", cate)

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
