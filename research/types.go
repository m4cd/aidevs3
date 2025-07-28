package main

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type Sample struct {
	Messages []Message `json:"messages"`
}

type AnswerType struct {
	Task   string  `json:"task"`
	ApiKey string  `json:"apikey"`
	Answer []string `json:"answer"`
}
