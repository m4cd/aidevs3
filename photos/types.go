package main

type Message struct {
	Task   string `json:"task"`
	Apikey string `json:"apikey"`
	Answer string `json:"answer"`
}
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Hints   string `json:"hints,omitempty"`
}

type UrlsJSON struct {
	Urls []string `json:"urls"`
}
