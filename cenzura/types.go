package main

type Payload struct {
	System string `json:"system"`
	Model string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool `json:"stream"`
}

type Response struct {
	Response string `json:"response"`
}

type Answer struct {
	Task string `json:"task"`
	ApiKey string `json:"apikey"`
	Answer string `json:"answer"`
}