package auth

import (
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
	"strings"
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

var currentAuthResponse *oAuthResponse

// GetAcessToken retorna o token de acesso atual. Se o token estiver expirado, ele tenta trocá-lo pelo refresh token.
// 1. Se não possui um token em memória, tenta pegar o último salvo em disco.
// 2. Se não conseguir, inicia o fluxo de autenticação pela primeira vez.
// 3. Se o token estiver expirado, tenta trocá-lo pelo refresh token.
func GetAcessToken() string {
	var err error

	if currentAuthResponse == nil {
		currentAuthResponse, err = GetSavedTokenFlow()
		if err != nil {
			currentAuthResponse = FirstTimeFlow()
		}
	}

	if tokenIsExpired() {
		currentAuthResponse, err = ExchangeRefreshToken(currentAuthResponse.RefreshToken)
		if err != nil {
			panic(err)
		}
	}

	return currentAuthResponse.AccessToken
}

func GetUserID() string {
	if currentAuthResponse == nil {
		GetAcessToken()
		return fmt.Sprintf("%d", currentAuthResponse.UserID)
	}
	return fmt.Sprintf("%d", currentAuthResponse.UserID)
}

func FirstTimeFlow() *oAuthResponse {
	SendAuthRequest()
	temp_token := getTempToken()
	if temp_token == "" {
		panic("Token não encontrado.")
	}
	authResponse, err := ExchangeCodeForToken(temp_token)
	if err != nil {
		panic(err)
	}
	return authResponse
}

func GetSavedTokenFlow() (*oAuthResponse, error) {
	authResponse, err := get()

	if err != nil {
		return nil, err
	}

	if authResponse.AccessToken == "" || authResponse.RefreshToken == "" {
		return nil, errors.New("token não encontrado")
	}

	return authResponse, nil
}

func save(authCredentials oAuthResponse) {
	f, err := os.Create("auth_response.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	as_json, _ := json.MarshalIndent(authCredentials, "", "\t")
	f.Write(as_json)
}

func get() (*oAuthResponse, error) {
	f, err := os.Open("auth_response.json")
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

func ExchangeRefreshToken(refresh_token string) (*oAuthResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", dotenv.Get("APP_ID"))

	data.Set("client_secret", dotenv.Get("APP_SECRET_KEY"))
	data.Set("refresh_token", refresh_token)

	req, err := http.NewRequest("POST", "https://api.mercadolibre.com/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)

		return nil, errors.New("Erro ao consumir refresh token: " + resp.Status + " body: " + bodyString)
	}

	jsonParser := json.NewDecoder(resp.Body)

	response := oAuthResponse{}

	err = jsonParser.Decode(&response)
	if err != nil {
		return nil, errors.New("Erro ao decodificar resposta:" + err.Error())
	}

	response.ExpirationDate = calculateExpirationDate(response.ExpiresIn)

	save(response)

	return &response, nil
}

func ExchangeCodeForToken(code string) (*oAuthResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", dotenv.Get("APP_ID"))

	data.Set("client_secret", dotenv.Get("APP_SECRET_KEY"))
	data.Set("code", code)
	data.Set("redirect_uri", dotenv.Get("REDIRECT_URI"))

	req, err := http.NewRequest("POST", "https://api.mercadolibre.com/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	jsonParser := json.NewDecoder(resp.Body)

	response := oAuthResponse{}

	err = jsonParser.Decode(&response)
	if err != nil {
		return nil, errors.New("Erro ao decodificar resposta:" + err.Error())
	}

	response.ExpirationDate = calculateExpirationDate(response.ExpiresIn)

	save(response)

	return &response, nil
}

func tokenIsExpired() bool {
	if currentAuthResponse == nil {
		return true
	}
	if currentAuthResponse.ExpirationDate.Before(time.Now().UTC()) {
		return true
	}
	return false
}

func calculateExpirationDate(expiresIn int) time.Time {
	nowTime := time.Now().UTC()
	return nowTime.Add(time.Duration(expiresIn) * time.Second)
}

// Aqui não é possível salvar o código, ele vai apenas pedir pra autenticar no navegador.
func SendAuthRequest() {
	client_id := dotenv.Get("APP_ID")
	redirect_uri := dotenv.Get("REDIRECT_URI")
	state := "12345"
	authPath := fmt.Sprintf("https://auth.mercadolivre.com.br/authorization?response_type=code&client_id=%s&redirect_uri=%s&state=%s", client_id, redirect_uri, state)

	openbrowser(authPath)
}

func getTempToken() string {
	fmt.Print("Cole o url aqui: ")
	var url string
	fmt.Scanln(&url)

	urlParts := strings.Split(url, "?")
	urlParts = strings.Split(urlParts[1], "&")

	for _, part := range urlParts {
		if strings.Contains(part, "code=") {
			code := strings.Split(part, "=")[1]
			return code
		}
	}
	return ""
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	}
	if err != nil {
		log.Fatal(err)
	}
}
