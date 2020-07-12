package http_authorizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"deepbooru"
)

type Authorizer struct {
	URL    string
	Client *http.Client
}

func New(url string) deepbooru.Authorizer {
	return &Authorizer{
		URL:    url,
		Client: http.DefaultClient,
	}
}

func (a *Authorizer) Close() {}

func (a *Authorizer) Authorize(credentials string) (deepbooru.Auth, error) {
	buff := new(bytes.Buffer)
	err := json.NewEncoder(buff).Encode(map[string]string{
		"credentials": credentials,
	})

	if err != nil {
		return deepbooru.Anonymous, err
	}

	req, err := http.NewRequest("POST", a.URL, buff)

	if err != nil {
		return deepbooru.Anonymous, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)

	if err != nil {
		return deepbooru.Anonymous, err
	}

	if resp.StatusCode == 403 || resp.StatusCode == 404 {
		return deepbooru.Anonymous, nil
	}

	if resp.StatusCode != 200 {
		return deepbooru.Anonymous, fmt.Errorf("Unknown error")
	}

	auth := deepbooru.Auth{}
	err = json.NewDecoder(resp.Body).Decode(&auth)

	if err != nil {
		return deepbooru.Anonymous, err
	}

	return auth, nil
}
