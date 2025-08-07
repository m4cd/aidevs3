package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	err := godotenv.Load("../.env")
	CheckErr(err)

	OpenAiApiKey := os.Getenv("OPENAI_API_KEY")
	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey),
	)
	Centrala := os.Getenv("URL_CNTRL")
	ApiKey := os.Getenv("API_KEY")
	TaskURL := Centrala + "data/" + ApiKey + "/notes.json"
	jpgFile := "image_full.png"

	tasks := GetTask(TaskURL)
	pdfText := ReadFileToString("text.txt")

	image, err := os.ReadFile(jpgFile)
	CheckErr(err)

	imageBase64 := base64.StdEncoding.EncodeToString(image)

	systemMessage := "Jesteś ekspertem specjalizującym się w odczytywaniu tekstu z przesłanego obrazu. Odpowiadaj jedynie tekstem wyekstrahowanym z obrazu (bez komentarzy i przemyśleń). Przesłany obraz zawiera odręczne notatki, więc miej to na uwadze."

	ChatCompletionContentPartImageParam := openai.ChatCompletionContentPartImageParam{
		ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
			URL:    fmt.Sprintf("data:image/jpeg;base64,%v", imageBase64),
			Detail: "high",
		},
		Type: "image_url",
	}

	params := openai.ChatCompletionNewParams{}
	params.Messages = append(params.Messages, openai.SystemMessage(systemMessage))
	params.Messages = append(params.Messages, openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
		{
			OfImageURL: &ChatCompletionContentPartImageParam,
		},
	}))
	params.Model = openai.ChatModelGPT4_1

	chatCompletion, err := openAiClient.Chat.Completions.New(
		context.TODO(),
		params,
	)
	CheckErr(err)

	imageText := chatCompletion.Choices[0].Message.Content

	context := "<kontekst>\n<pdf>\n" + pdfText + "\n</pdf>\n"
	context += "<tekst z obrazka>\n"
	context += imageText + "\n"
	context += "</tekst z obrazka>\n"
	context += "</kontekst>\n"

	fmt.Println("### KONTEKST ###")

	systemMessage = ReadFileToString("systemprompt.txt")
	userMessage := context
	response := CompleteChatJson(openAiClient, userMessage, tasks, systemMessage)

	fmt.Println(response)

	var ans AnswerType
	ans.ApiKey = ApiKey
	ans.Task = "notes"

	var jsonAnswer map[string]interface{}
	json.Unmarshal([]byte(response), &jsonAnswer)
	ans.Answer = jsonAnswer

	rsp := SendAnswer(ans, Centrala+"report")
	fmt.Println(string(rsp))

}
