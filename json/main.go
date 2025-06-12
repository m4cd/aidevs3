package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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
	URL := os.Getenv("URL_CNTRL") + "report"
	ApiKey := os.Getenv("API_KEY")
	jsonFileName := "json.txt"

	jsonFileContents, err := os.ReadFile(jsonFileName)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	var jsonFile JsonFile
	err = json.Unmarshal(jsonFileContents, &jsonFile)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}

	for i, v := range jsonFile.TestData {

		var nums [2]int64
		_, err := fmt.Sscanf(v.Question, "%d + %d", &nums[0], &nums[1])
		if err != nil {
			log.Fatal("Error during fmt.Sscanf(): ", err)
		}

		jsonFile.TestData[i].Answer = nums[0] + nums[1]

		if v.Test != nil {
			openAiClient := openai.NewClient(
				option.WithAPIKey(OpenAiApiKey), // defaults to os.LookupEnv("OPENAI_API_KEY")
			)

			chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage(fmt.Sprintf("Answer this question as precisely and shortly as possible. Do not elaborate. Omit punctuation. Question: %s", v.Test.Q)),
				},
				Model: openai.ChatModelGPT4o,
			})
			if err != nil {
				panic(err.Error())
			}
			Answer := chatCompletion.Choices[0].Message.Content
			jsonFile.TestData[i].Test.A = Answer

		}
	}

	var JsonAnswer JsonAnswer
	JsonAnswer.ApiKey = ApiKey
	JsonAnswer.Task = "JSON"
	JsonAnswer.Answer = jsonFile
	JsonAnswer.Answer.ApiKey = ApiKey
	


	resp := SendAnser(JsonAnswer, URL)
	fmt.Println(resp.Code)
	fmt.Println(resp.Message)

}

func SendAnser(ans JsonAnswer, URL string) ResponseMessage {
	httpClient := http.Client{}

	jsonBytes, err := json.Marshal(ans)
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

	var respJson ResponseMessage
	bodyAnswerBytes, _ := io.ReadAll(res.Body)
	json.Unmarshal(bodyAnswerBytes, &respJson)

	return respJson
}
