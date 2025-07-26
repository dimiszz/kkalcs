// requests.go (Modified)
package requests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"dimi/kkalcs/shpeapi/auth" // Your existing auth package for Shopee
)

var USER_ID string // This seems to be for Mercado Libre. For Shopee, we'll use shopID.

type Method string

const (
	GET    Method = http.MethodGet
	POST   Method = http.MethodPost
	PUT    Method = http.MethodPut
	DELETE Method = http.MethodDelete
)

// QueryParams is a helper for building URL query parameters.
type QueryParams map[string]string

// Add adds a key-value pair to the query parameters.
func (qp QueryParams) Add(key, value string) {
	qp[key] = value
}

// Encode converts the QueryParams to a URL-encoded string.
func (qp QueryParams) Encode() string {
	values := url.Values{}
	for k, v := range qp {
		values.Add(k, v)
	}
	return values.Encode()
}

// NewQueryParams creates a new QueryParams map.
func NewQueryParams() QueryParams {
	return make(QueryParams)
}

// MakeRequest (Your existing Mercado Libre focused function)
func MakeRequest(method Method, url string, body *bytes.Buffer) (*http.Response, error) {
	slog.Debug("Making request", "method", method, "url", url)
	var bodyReader io.Reader
	if body != nil {
		bodyReader = body
	} else {
		bodyReader = nil
	}

	accessToken := auth.GetAcessToken() // Mercado Libre access token
	var bearer = "Bearer " + accessToken

	req, err := http.NewRequest(string(method), url, bodyReader)
	if err != nil {
		panic(err) // Consider returning error instead of panic
	}

	req.Header.Add("Authorization", bearer)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		slog.Debug("Request failed ", "Response Body:", string(bodyBytes))
		return nil, fmt.Errorf("error: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// MakeSimpleRequest (Your existing Mercado Libre focused function)
func MakeSimpleRequest(method Method, url string, body *bytes.Buffer) ([]byte, error) {
	resp, err := MakeRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer requisição: %s", err)
	}
	bodybyte, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o corpo da resposta: %s", err)
	}
	return bodybyte, nil
}

// MakeShopeeRequest makes a request to the Shopee API, handling authentication and signing.
func MakeShopeeRequest(method Method, baseURL, apiPath string, queryParams QueryParams, body interface{}) ([]byte, error) {
	partnerIDStr := auth.GetPartnerID()
	//partnerID, err := strconv.ParseInt(partnerIDStr, 10, 64)
	// if err != nil {
	// 	return nil, fmt.Errorf("invalid Shopee Partner ID: %w", err)
	// }

	shopIDStr := auth.GetUserID() // Your auth.GetUserID() returns the shop_id as string for Shopee
	shopID, err := strconv.ParseInt(shopIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid Shopee Shop ID: %w", err)
	}

	accessToken := auth.GetAcessToken() // This gets the Shopee access token

	timestamp := time.Now().Unix()

	// Build the base string for signing
	// Format: partner_id + API_path + timestamp + access_token + shop_id (if available and relevant)
	// For API endpoints that require authentication (most of them), access_token and shop_id are part of the base string.
	// Check Shopee docs for specific endpoint signing rules.
	baseString := fmt.Sprintf("%s%s%d%s%d", partnerIDStr, apiPath, timestamp, accessToken, shopID)
	sign := auth.CalculateHmacSha256(baseString, auth.GetPartnerKey()) // Using your existing auth.CalculateHmacSha256

	// Add common Shopee parameters to query params
	if queryParams == nil {
		queryParams = NewQueryParams()
	}
	queryParams.Add("partner_id", partnerIDStr)
	queryParams.Add("timestamp", strconv.FormatInt(timestamp, 10))
	queryParams.Add("sign", sign)
	queryParams.Add("shop_id", strconv.FormatInt(shopID, 10)) // Shop ID is almost always required for shop-level APIs
	queryParams.Add("access_token", accessToken)              // Access token usually in query for GET, sometimes in body for POST/refresh

	fullURL := fmt.Sprintf("%s%s?%s", baseURL, apiPath, queryParams.Encode())
	slog.Debug("Making Shopee request", "method", method, "url", fullURL)

	var reqBody io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body for Shopee: %w", err)
		}
		reqBody = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequest(string(method), fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create Shopee request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send Shopee request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Shopee API error: status code %d, body: %s", resp.StatusCode, string(respBodyBytes))
	}

	return respBodyBytes, nil
}

func UrlValuesToJson(body url.Values) ([]byte, error) {
	m := make(map[string]string)
	for key, vals := range body {
		if len(vals) > 0 {
			m[key] = vals[0] // Get the first value of each key
		}
	}

	// Convert the map to JSON
	jsonBody, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error converting values to JSON: %v", err)
	}
	return jsonBody, nil
}
