package main

type AnswerType struct {
	Task   string                 `json:"task"`
	ApiKey string                 `json:"apikey"`
	Answer map[string]interface{} `json:"answer"`
}

type LlmResponseType struct {
	Thinking  string   `json:"thinking"`
	HasAnswer bool     `json:"has_answer"`
	Answer    string   `json:"answer"`
	NextUrls  []string `json:"next_urls"`
}
