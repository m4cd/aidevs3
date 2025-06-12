package main

type Test struct {
	Q string `json:"q"`
	A string `json:"a"`
}

type TestData struct {
	Question string `json:"question"`
	Answer   int64  `json:"answer"`
	Test     *Test  `json:"test"`
}

type JsonFile struct {
	ApiKey      string     `json:"apikey"`
	Description string     `json:"description"`
	Copyright   string     `json:"copyright"`
	TestData    []TestData `json:"test-data"`
}

type JsonAnswer struct {
	Task   string   `json:"task"`
	ApiKey string   `json:"apikey"`
	Answer JsonFile `json:"answer"`
}

type ResponseMessage struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

