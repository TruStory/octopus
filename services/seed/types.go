package main

// MockedRegisterResponse represents a newly registeres mock user
type MockedRegisterResponse struct {
	Data struct {
		UserID               string `json:"userId"`
		Username             string `json:"username"`
		FullName             string `json:"fullname"`
		Address              string `json:"address"`
		AuthenticationCookie string `json:"authenticationCookie"`
	}
}
