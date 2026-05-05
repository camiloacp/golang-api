package main

import (
	"fmt"
	"io"
	"net/http"
)

const (
	BaseURL = "http://localhost:8080"
)

func main() {
	lc := loginClient(BaseURL+"/v1/login", "camilo@test.com", "secreto123")
	//fmt.Println("Token:", lc.Data.Token)

	person := Person{
		Name: "Diego Cortés",
		Age:  30,
		Communities: []Community{
			{Name: "Devops"},
			{Name: "Gym"},
			{Name: "Ciclismo"},
		},
	}
	cp := createPerson(BaseURL+"/v1/persons", lc.Data.Token, person)
	fmt.Println("Create person:", cp.Message)
}

func httpClient(method, url, contentType, token string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	return resp, nil
}
