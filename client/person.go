package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func createPerson(url, token string, person Person) GeneralResponse {
	data := bytes.NewBuffer(nil)
	if err := json.NewEncoder(data).Encode(person); err != nil {
		log.Fatalf("error encoding person: %v", err)
	}

	resp, err := httpClient(http.MethodPost, url, "application/json", token, data)
	if err != nil {
		log.Fatalf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("error status code: %d body: %s", resp.StatusCode, body)
	}

	dataResponse := GeneralResponse{}
	if err := json.Unmarshal(body, &dataResponse); err != nil {
		log.Fatalf("error unmarshalling response: %v", err)
	}

	return dataResponse
}
