package main

type AnswerType struct {
	Task   string                 `json:"task"`
	ApiKey string                 `json:"apikey"`
	Answer map[string]interface{} `json:"answer"`
}
