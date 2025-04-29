package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"dimi/kkalcs/mlapi/requests"
)

type TestUserResponse struct {
	ID       int    `json:"id"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
	SiteID   string `json:"site_id"`
}

// Função usada para criar usuários de teste.
func CreateTestUser() (*TestUserResponse, error) {
	request_url := "https://api.mercadolibre.com/users/test_user"

	body_request := url.Values{}

	body_request.Set("site_id", "MLB")

	body_json, err := requests.UrlValuesToJson(body_request)
	if err != nil {
		return nil, err
	}

	body_response, err := requests.MakeSimpleRequest(requests.POST, request_url, bytes.NewBuffer(body_json))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %v", err)
	}

	var user TestUserResponse
	err = json.Unmarshal(body_response, &user)
	if err != nil {
		return nil, fmt.Errorf("erro ao interpretar resposta: %v", err)
	}

	fmt.Println("User:", user)

	return &user, nil
}

func CreateListings() error {
	userID := "2408860744"
	err := createListingForSeller(userID)
	if err != nil {
		return err
	}
	return nil
}

func createListingForSeller(userID string) error {
	// Criar uma publicação para esse vendedor
	// O vendedor precisa de pelo menos um produto para ser considerado vendedor ativo
	// Use a API de publicações para criar o produto

	// Exemplo de dados do produto (substitua com os dados reais)
	product := map[string]interface{}{
		"site_id":            "MLB",
		"title":              "Item de Teste – Por favor, NÃO OFERTAR!",
		"category_id":        "MLA3025", // Defina a categoria do produto
		"price":              100.0,
		"currency_id":        "BRL",
		"available_quantity": 10,
		"condition":          "new",
		"listing_type_id":    "bronze",
		"description":        "Descrição do produto de teste",
		"tags":               []string{"test_item"},
	}
	request_url := "https://api.mercadolibre.com/items"

	product_json, err := json.Marshal(product)
	if err != nil {
		return err
	}

	// Endpoint para criar a publicação

	// Faz a requisição para criar a publicação
	resp, err := requests.MakeRequest(requests.POST, request_url, bytes.NewBuffer(product_json))
	if err != nil {
		fmt.Println("Erro ao criar publicação:", err)
		return err
	}
	bodybyte, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("erro ao ler o corpo da resposta: %s", err)
	}
	// Exibe a resposta da criação da publicação
	fmt.Println("Produto publicado com sucesso:", string(bodybyte))
	return nil
}
