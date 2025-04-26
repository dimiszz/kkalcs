package requests

import (
	"dimi/ml-auth/auth"
	"fmt"
	"io"
	"net/http"
)

func MakeRequest(url string, body io.Reader) (*http.Response, error) {
	access_token := auth.GetAcessToken()
	var bearer = "Bearer " + access_token

	req, err := http.NewRequest("GET", url, body)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", bearer)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error: status code %d", resp.StatusCode)
	}

	return resp, nil
}

func MakeSimpleRequest(url string, body io.Reader) ([]byte, error) {
	resp, err := MakeRequest(url, body)
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
