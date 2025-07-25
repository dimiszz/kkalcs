package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"dimi/kkalcs/dotenv"
)

type oAuthResponse struct {
	AccessToken    string    `json:"access_token"`
	TokenType      string    `json:"token_type"`
	ExpiresIn      int       `json:"expires_in"`
	ExpirationDate time.Time `json:"expiration_date"`
	Scope          string    `json:"scope"`
	UserID         int       `json:"user_id"`
	RefreshToken   string    `json:"refresh_token"`
}

// ShopeeAuthResponse handles the different JSON structures from Shopee's API.
type ShopeeAuthResponse struct {
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	ExpiresIn      int       `json:"expires_in"`
	ShopIDList     []int64   `json:"shop_id_list"` // For initial token exchange
	ShopID         int64     `json:"shop_id"`      // For refresh token exchange
	PartnerID      int64     `json:"partner_id"`
	MerchantID     int64     `json:"merchant_id,omitempty"`
	ExpirationDate time.Time `json:"-"`
}

// ToOAuthResponse converts the Shopee-specific response to the internal oAuthResponse struct.
func (s *ShopeeAuthResponse) ToOAuthResponse() *oAuthResponse {
	var shopID int64
	if len(s.ShopIDList) > 0 {
		shopID = s.ShopIDList[0]
	} else if s.ShopID > 0 {
		shopID = s.ShopID
	}

	return &oAuthResponse{
		AccessToken:    s.AccessToken,
		RefreshToken:   s.RefreshToken,
		ExpiresIn:      s.ExpiresIn,
		ExpirationDate: s.ExpirationDate,
		UserID:         int(shopID),
		TokenType:      "Bearer",
		Scope:          "",
	}
}

var currentAuthResponse *oAuthResponse

// GetAcessToken returns the current access token, handling token refresh and initial auth flows.
func GetAcessToken() string {
	var err error

	if currentAuthResponse == nil {
		currentAuthResponse, err = GetSavedTokenFlow()
		if err != nil {
			currentAuthResponse = FirstTimeFlow()
		}
	}

	if tokenIsExpired() {
		shopID := int64(currentAuthResponse.UserID)
		currentAuthResponse, err = ExchangeRefreshToken(currentAuthResponse.RefreshToken, shopID)
		if err != nil {
			panic(err)
		}
	}

	return currentAuthResponse.AccessToken
}

// GetUserID returns the shop_id, ensuring the token flow has been initiated.
func GetUserID() string {
	if currentAuthResponse == nil {
		GetAcessToken()
	}
	return fmt.Sprintf("%d", currentAuthResponse.UserID)
}

// FirstTimeFlow handles the entire initial authentication process.
func FirstTimeFlow() *oAuthResponse {
	SendAuthRequest()
	code, shopIDStr, err := getTempToken()
	if err != nil {
		panic(err)
	}

	shopID, err := strconv.ParseInt(shopIDStr, 10, 64)
	if err != nil {
		panic("Invalid shop_id received from URL: " + shopIDStr)
	}

	authResponse, err := ExchangeCodeForToken(code, shopID)
	if err != nil {
		panic(err)
	}
	return authResponse
}

// GetSavedTokenFlow attempts to load the last saved token from disk.
func GetSavedTokenFlow() (*oAuthResponse, error) {
	authResponse, err := get()
	if err != nil {
		return nil, err
	}

	if authResponse.AccessToken == "" || authResponse.RefreshToken == "" {
		return nil, errors.New("saved token is incomplete")
	}

	return authResponse, nil
}

// save persists the authentication credentials to a JSON file.
func save(authCredentials oAuthResponse) {
	f, err := os.Create("auth_response-shpe.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	as_json, _ := json.MarshalIndent(authCredentials, "", "\t")
	_, err = f.Write(as_json)
	if err != nil {
		panic(err)
	}
}

// get retrieves the authentication credentials from the JSON file.
func get() (*oAuthResponse, error) {
	f, err := os.Open("auth_response-shpe.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var authResponse oAuthResponse
	err = json.Unmarshal(data, &authResponse)
	if err != nil {
		return nil, err
	}
	return &authResponse, nil
}

// ExchangeCodeForToken gets the initial token set.
func ExchangeCodeForToken(code string, shopID int64) (*oAuthResponse, error) {
	timestamp := time.Now().Unix()
	path := "/api/v2/auth/token/get"
	partnerID := dotenv.Get("APP_ID_SHP")
	partnerKey := dotenv.Get("APP_SECRET_KEY_SHP")

	type requestBody struct {
		Code   string `json:"code"`
		ShopID int64  `json:"shop_id"`
	}

	bodyData := requestBody{
		Code:   code,
		ShopID: shopID,
	}
	bodyBytes, err := json.Marshal(bodyData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	baseString := fmt.Sprintf("%s%s%d", partnerID, path, timestamp)
	sign := CalculateHmacSha256(baseString, partnerKey)

	baseURL := "https://partner.shopeemobile.com"
	fullURL := fmt.Sprintf("%s%s?partner_id=%s&timestamp=%d&sign=%s", baseURL, path, partnerID, timestamp, sign)

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("Shopee API Raw Response (Initial Token):", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}

	var shopeeResponse ShopeeAuthResponse
	if err := json.Unmarshal(respBody, &shopeeResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	shopeeResponse.ExpirationDate = calculateExpirationDate(shopeeResponse.ExpiresIn)
	response := shopeeResponse.ToOAuthResponse()
	save(*response)

	return response, nil
}

// ExchangeRefreshToken refreshes an expired access token.
func ExchangeRefreshToken(refreshToken string, shopID int64) (*oAuthResponse, error) {
	timestamp := time.Now().Unix()
	path := "/api/v2/auth/access_token/get"
	partnerID := dotenv.Get("APP_ID_SHP")
	partnerKey := dotenv.Get("APP_SECRET_KEY_SHP")

	type requestBody struct {
		RefreshToken string `json:"refresh_token"`
		ShopID       int64  `json:"shop_id"`
		PartnerID    int64  `json:"partner_id"`
	}

	partnerIDInt, _ := strconv.ParseInt(partnerID, 10, 64)
	bodyData := requestBody{
		RefreshToken: refreshToken,
		ShopID:       shopID,
		PartnerID:    partnerIDInt,
	}
	bodyBytes, err := json.Marshal(bodyData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	baseString := fmt.Sprintf("%s%s%d", partnerID, path, timestamp)
	sign := CalculateHmacSha256(baseString, partnerKey)

	baseURL := "https://partner.shopeemobile.com"
	fullURL := fmt.Sprintf("%s%s?partner_id=%s&timestamp=%d&sign=%s", baseURL, path, partnerID, timestamp, sign)

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("Shopee API Raw Response (Refresh):", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}

	var shopeeResponse ShopeeAuthResponse
	if err := json.Unmarshal(respBody, &shopeeResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if shopeeResponse.RefreshToken == "" {
		shopeeResponse.RefreshToken = refreshToken
	}
	shopeeResponse.ExpirationDate = calculateExpirationDate(shopeeResponse.ExpiresIn)
	response := shopeeResponse.ToOAuthResponse()
	save(*response)

	return response, nil
}

// tokenIsExpired checks if the current token has passed its expiration time.
func tokenIsExpired() bool {
	if currentAuthResponse == nil {
		return true
	}
	return currentAuthResponse.ExpirationDate.Before(time.Now().UTC())
}

// calculateExpirationDate determines the token's expiry time.
func calculateExpirationDate(expiresIn int) time.Time {
	return time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
}

// SendAuthRequest generates the initial authorization URL and opens it in the browser.
func SendAuthRequest() {
	partnerID := dotenv.Get("APP_ID_SHP")
	partnerKey := dotenv.Get("APP_SECRET_KEY_SHP")
	redirectURI := dotenv.Get("REDIRECT_URI_SHP")
	timestamp := time.Now().Unix()
	path := "/api/v2/shop/auth_partner"

	// FIX: Updated to use the correct signature generation method.
	baseString := fmt.Sprintf("%s%s%d", partnerID, path, timestamp)
	sign := CalculateHmacSha256(baseString, partnerKey)

	authPath := fmt.Sprintf("https://partner.shopeemobile.com%s?partner_id=%s&redirect=%s&timestamp=%d&sign=%s", path, partnerID, redirectURI, timestamp, sign)
	openbrowser(authPath)
}

// CalculateHmacSha256 computes the HMAC-SHA256 signature for a given base string and key.
func CalculateHmacSha256(baseString string, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(baseString))
	return hex.EncodeToString(h.Sum(nil))
}

// getTempToken prompts the user to paste the redirect URL and extracts the code and shop_id.
func getTempToken() (code string, shopID string, err error) {
	fmt.Print("Paste the redirect URL here: ")
	var redirectedURL string
	fmt.Scanln(&redirectedURL)

	parsedURL, err := url.Parse(redirectedURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	queryParams := parsedURL.Query()
	code = queryParams.Get("code")
	shopID = queryParams.Get("shop_id")

	if code == "" || shopID == "" {
		return "", "", errors.New("could not find 'code' or 'shop_id' in the provided URL")
	}

	return code, shopID, nil
}

// openbrowser opens a URL in the default web browser.
func openbrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}
