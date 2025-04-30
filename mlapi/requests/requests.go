package requests

import (
	"bytes"
	"dimi/kkalcs/mlapi/auth"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

var USER_ID string

type Method string

const (
	GET    Method = http.MethodGet
	POST   Method = http.MethodPost
	PUT    Method = http.MethodPut
	DELETE Method = http.MethodDelete
)

func MakeRequest(method Method, url string, body *bytes.Buffer) (*http.Response, error) {
	slog.Debug("Making request", "method", method, "url", url)
	var bodyReader io.Reader
	if body != nil {
		bodyReader = body
	} else {
		bodyReader = nil
	}

	access_token := auth.GetAcessToken()
	var bearer = "Bearer " + access_token

	req, err := http.NewRequest(string(method), url, bodyReader)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", bearer)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		slog.Debug("Request failed ", "Response Body:", string(body))
		return nil, fmt.Errorf("error: status code %d", resp.StatusCode)
	}

	return resp, nil
}

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

func UrlValuesToJson(body url.Values) ([]byte, error) {
	m := make(map[string]string)
	for key, vals := range body {
		if len(vals) > 0 {
			m[key] = vals[0] // Pega o primeiro valor de cada chave
		}
	}

	// Converte o map para JSON
	jsonBody, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("erro ao converter valores para JSON: %v", err)
	}
	return jsonBody, nil
}
