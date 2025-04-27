package test

import (
	"encoding/json"
	"fmt"
	"net/url"

	"dimi/kkalcs/mlapi/requests"
)

type TestUserResponse struct {
	ID       int    `json:"id"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
	SiteID   string `json:"site_id"`
}

func CreateTestUser() (*TestUserResponse, error) {
	request_url := "https://api.mercadolibre.com/users/test_user"

	body_request := url.Values{}

	body_request.Set("site_id", "MLB")

	body_response, err := requests.MakeSimpleRequest(requests.POST, request_url, body_request)
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
