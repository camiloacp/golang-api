package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"golang-api/model"
)

func loginClient(url, email, password string) LoginResponse {
	login := model.Login{
		Email:    email,
		Password: password,
	}

	data := bytes.NewBuffer(nil)
	if err := json.NewEncoder(data).Encode(login); err != nil {
		log.Fatalf("error encoding login: %v", err)
	}

	resp, err := httpClient(http.MethodPost, url, "application/json", "", data)
	if err != nil {
		log.Fatalf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("error status code: %d body: %s", resp.StatusCode, body)
	}

	dataResponse := LoginResponse{}
	err = json.Unmarshal(body, &dataResponse)
	if err != nil {
		log.Fatalf("error unmarshalling response: %v", err)
	}

	return dataResponse
}
