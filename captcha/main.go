package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error while loading .env file.")
	}
	OpenAiApiKey := os.Getenv("OPENAI_API_KEY")
	URL := os.Getenv("URL")
	LOGIN := os.Getenv("LOGIN")
	PASSWORD := os.Getenv("PASSWORD")

	httpClient := http.Client{}

	// GET the task
	taskResponse, err := httpClient.Get(URL)
	if err != nil {
		fmt.Printf("Cannot get task response: %s\n", err)
		os.Exit(1)
	}
	defer taskResponse.Body.Close()

	// Parsing task and asking LLM to answer

	doc, err := goquery.NewDocumentFromReader(taskResponse.Body)
	if err != nil {
		log.Fatal(err)
	}

	q := doc.Find("p#human-question").Each(func(index int, item *goquery.Selection) {})
	Question := string(q.Text()[9:])

	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey), // defaults to os.LookupEnv("OPENAI_API_KEY")
	)

	chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(fmt.Sprintf("Answer this question as precisely and shortly as possible. Do not elaborate. Omit punctuation. Question: %s", Question)),
		},
		Model: openai.ChatModelGPT4o,
	})
	if err != nil {
		panic(err.Error())
	}
	Answer := chatCompletion.Choices[0].Message.Content

	// Sending answer

	body := fmt.Sprintf(`username=%s&password=%s&answer=%s`, LOGIN, PASSWORD, Answer)

	bodyBytes := []byte(body)
	bodyReader := bytes.NewReader(bodyBytes)

	req, err := http.NewRequest(http.MethodPost, URL, bodyReader)
	if err != nil {
		fmt.Printf("Cannot create POST request: %s\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := httpClient.Do(req)

	if err != nil {
		fmt.Printf("Client error making http request: %s\n", err)
		os.Exit(1)
	}

	defer res.Body.Close()

	resBytes, _ := io.ReadAll(res.Body)
	fmt.Println(string(resBytes))

}
