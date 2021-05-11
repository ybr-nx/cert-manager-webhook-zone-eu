package main

type ApiConfig struct {
	ApiUserName,  ApiKey, ZoneName, ApiUrl string
}

type TxtRecordResponse struct {
	Records []TxtRecord `json:"data"`
	Message   string    `json:"message"`
}

type TxtRecord struct {
	Id          string `json:"id"`
	ResourceUrl string `json:"resource_url"`
	Name        string `json:"name"`
	Destination string `json:"destination"`
}

type Verification struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}