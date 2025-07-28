package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error while loading .env file.")
	}
	OpenAiFtModel := os.Getenv("OPENAI_FT_MODEL")
	OpenAiApiKey := os.Getenv("OPENAI_API_KEY")
	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey),
	)

	ApiKey := os.Getenv("API_KEY")
	Centrala := os.Getenv("URL_CNTRL")

	verifyFilepath := "lab_data/verify.txt"

	var finalAnswer AnswerType
	finalAnswer.ApiKey = ApiKey
	finalAnswer.Task = "research"

	DataAlreadyPrepared := true

	if !DataAlreadyPrepared {
		fmt.Println("Preparing data for training...")
		correctFilepath := "lab_data/correct.txt"
		incorrectFilepath := "lab_data/incorect.txt"
		correctJSON := "lab_data/correct.json"
		incorrectJSON := "lab_data/incorect.json"

		PrepareData(correctFilepath, correctJSON, incorrectFilepath, incorrectJSON)
	}

	file, err := os.Open(verifyFilepath)
	if err != nil {
		panic(err.Error())
	}
	fileScanner := bufio.NewScanner(file)

	for fileScanner.Scan() {
		input := strings.Split(fileScanner.Text(), "=")
		num := input[0]
		words := input[1]
		systemMessage := "Clasify input based on your training."
		userMessage := words

		chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(userMessage),
				openai.SystemMessage(systemMessage),
			},
			Model:       OpenAiFtModel,
			Temperature: openai.Float(0.0),
		})
		if err != nil {
			panic(err.Error())
		}
		if chatCompletion.Choices[0].Message.Content == "1" {
			finalAnswer.Answer = append(finalAnswer.Answer, num)
		}

	}

	fmt.Println("ANSWER:")
	fmt.Println(finalAnswer.Answer)
	fmt.Println("=================")

	ansResp := SendAnswer(finalAnswer, Centrala+"report")
	fmt.Println(string(ansResp))

}

func PrepareData(correctFilepath string, correctJSON string, incorrectFilepath string, incorrectJSON string) {
	correctSamples := []Sample{}
	incorrectSamples := []Sample{}

	// correct
	file, err := os.Open(correctFilepath)
	if err != nil {
		panic(err.Error())
	}
	fileScanner := bufio.NewScanner(file)

	for fileScanner.Scan() {
		var smpl Sample
		smpl.Messages = append(smpl.Messages, Message{
			Role:    "system",
			Content: "validate data",
		})
		smpl.Messages = append(smpl.Messages, Message{
			Role:    "user",
			Content: fileScanner.Text(),
		})
		smpl.Messages = append(smpl.Messages, Message{
			Role:    "assistant",
			Content: "1",
		})
		correctSamples = append(correctSamples, smpl)
	}
	correctSamplesPrintable, _ := json.MarshalIndent(correctSamples, "", " ")
	if err := os.WriteFile(correctJSON, correctSamplesPrintable, 0666); err != nil {
		log.Fatal(err)
	}
	file.Close()

	// incorrect
	file, err = os.Open(incorrectFilepath)
	if err != nil {
		panic(err.Error())
	}
	fileScanner = bufio.NewScanner(file)

	for fileScanner.Scan() {
		var smpl Sample
		smpl.Messages = append(smpl.Messages, Message{
			Role:    "system",
			Content: "validate data",
		})
		smpl.Messages = append(smpl.Messages, Message{
			Role:    "user",
			Content: fileScanner.Text(),
		})
		smpl.Messages = append(smpl.Messages, Message{
			Role:    "assistant",
			Content: "0",
		})
		incorrectSamples = append(incorrectSamples, smpl)
	}
	incorrectSamplesPrintable, _ := json.MarshalIndent(incorrectSamples, "", " ")
	if err := os.WriteFile(incorrectJSON, incorrectSamplesPrintable, 0666); err != nil {
		log.Fatal(err)
	}
	file.Close()
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
