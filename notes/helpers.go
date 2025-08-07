package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	"github.com/openai/openai-go/shared/constant"
)

type AnswerType struct {
	Task   string                 `json:"task"`
	ApiKey string                 `json:"apikey"`
	Answer map[string]interface{} `json:"answer"`
}

func SendAnswer(answer AnswerType, URL string) []byte {
	httpClient := http.Client{}

	jsonBytes, err := json.Marshal(answer)
	if err != nil {
		fmt.Printf("Cannot marshal json: %s\n", err)
		os.Exit(1)
	}

	bodyReader := bytes.NewReader(jsonBytes)

	ansReq, err := http.NewRequest(http.MethodPost, URL, bodyReader)
	if err != nil {
		fmt.Printf("Cannot create response request: %s\n", err)
		os.Exit(1)
	}

	ansReq.Header.Set("Content-Type", "application/json")
	ansReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:138.0) Gecko/20100101 Firefox/138.0")

	res, err := httpClient.Do(ansReq)

	if err != nil {
		fmt.Printf("Client error making http request: %s\n", err)
		os.Exit(1)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		fmt.Printf("UnAuthorized with code: %d\n", res.StatusCode)
		os.Exit(1)
	}

	bodyAnswerBytes, _ := io.ReadAll(res.Body)

	return bodyAnswerBytes
}

func CompleteChatJson(openAiClient openai.Client, kontekst string, questions map[string]interface{}, systemMessage string) string {

	q := ""
	for k, v := range questions {
		q += fmt.Sprintf("\"%v\":\"%v\",\n", k, v)
	}
	fmt.Println(q)

	chatCompletion, err := openAiClient.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(kontekst),
			openai.UserMessage(q),
			openai.SystemMessage(systemMessage),
		},
		Model:       openai.ChatModelGPT4o,
		Temperature: openai.Float(0.1),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{
				Type: constant.JSONObject("json_object"),
			},
		},
	})

	if err != nil {
		panic(err.Error())
	}

	return chatCompletion.Choices[0].Message.Content
}

func GetTask(TaskURL string) map[string]interface{} {
	var tasks map[string]interface{}

	httpClient := http.Client{}
	taskResponse, err := httpClient.Get(TaskURL)
	if err != nil {
		fmt.Printf("Cannot get task response: %s\n", err)
		os.Exit(1)
	}
	defer taskResponse.Body.Close()

	taskBody, _ := io.ReadAll(taskResponse.Body)
	json.Unmarshal(taskBody, &tasks)

	return tasks
}

func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}

func ReadFileToString(filename string) string {
	b, err := os.ReadFile(filename)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	return string(b)
}