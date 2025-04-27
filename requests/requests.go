package requests

import (
	"bytes"
	"dimi/ml-auth/auth"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Method string

const (
	GET    Method = "GET"
	POST   Method = "POST"
	PUT    Method = "PUT"
	DELETE Method = "DELETE"
)

func MakeRequest(method Method, url string, body url.Values) (*http.Response, error) {

	var body_reader *bytes.Buffer
	if body != nil {
		converted, _ := urlValuesToJson(body)
		body_reader = bytes.NewBuffer(converted)
	} else {
		body_reader = bytes.NewBuffer(nil)
	}

	access_token := auth.GetAcessToken()
	var bearer = "Bearer " + access_token

	req, err := http.NewRequest(string(method), url, body_reader)
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
		return nil, fmt.Errorf("error: status code %d", resp.StatusCode)
	}

	return resp, nil
}

func MakeSimpleRequest(method Method, url string, body url.Values) ([]byte, error) {

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

func urlValuesToJson(body url.Values) ([]byte, error) {
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
