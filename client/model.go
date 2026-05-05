package main

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GeneralResponse struct {
	MessageType string `json:"message_type"`
	Message     string `json:"message"`
}

// LoginResponse
type LoginResponse struct {
	GeneralResponse
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

type Community struct {
	Name string `json:"name"`
}

type Communities []Community

type Person struct {
	Name        string      `json:"name"`
	Age         int         `json:"age"`
	Communities Communities `json:"communities"`
}
