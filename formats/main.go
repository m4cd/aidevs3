package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Categories struct {
	People   []string `json:"people"`
	Hardware []string `json:"hardware"`
}

type Answer struct {
	Task   string     `json:"task"`
	ApiKey string     `json:"apikey"`
	Answer Categories `json:"answer"`
}

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error while loading .env file.")
	}

	ApiKey := os.Getenv("API_KEY")
	var ans Answer
	ans.Task = "kategorie"
	ans.ApiKey = ApiKey

	filesDir := "./pliki"
	OpenAiApiKey := os.Getenv("OPENAI_API_KEY")
	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey),
	)

	files, err := os.ReadDir(filesDir)
	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}

	for _, file := range files {
		filePath := filesDir + "/" + file.Name()
		content := ExtractFileContents(filePath, openAiClient)

		if content != "" {
			cat := CategorizeContents(content, openAiClient)

			if cat == "people" {
				ans.Answer.People = append(ans.Answer.People, file.Name())
			} else if cat == "hardware" {
				ans.Answer.Hardware = append(ans.Answer.Hardware, file.Name())
			}
		}

	}

	Centrala := os.Getenv("URL_CNTRL")
	ansRep := SendAnswer(ans, Centrala+"report")
	fmt.Println(string(ansRep))

}

func ExtractFileContents(filePath string, openAiClient openai.Client) string {
	var res string
	switch path.Ext(filePath) {
	case ".txt":
		b, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		res = string(b)
	case ".png":
		res = "png"
		image, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println("Error while loading image file.")
			os.Exit(1)
		}
		imageBase64 := base64.StdEncoding.EncodeToString(image)

		userMessage := `You are a specialist in optical character recognition. Analyze the image upleaded and return the exact text from the image. Return only the text extracted from the image and nothing else.`
		ChatCompletionContentPartImageParam := openai.ChatCompletionContentPartImageParam{
			ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
				URL:    fmt.Sprintf("data:image/jpeg;base64,%v", imageBase64),
				Detail: "high",
			},
			Type: "image_url",
		}

		params := openai.ChatCompletionNewParams{}
		params.Messages = append(params.Messages, openai.UserMessage(userMessage))
		params.Messages = append(params.Messages, openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			{
				OfImageURL: &ChatCompletionContentPartImageParam,
			},
		}))
		params.Model = openai.ChatModelGPT4o
		// params.Temperature = param.Opt[float64]{Value: 0.0}

		chatCompletion, err := openAiClient.Chat.Completions.New(
			context.TODO(),
			params,
		)
		if err != nil {
			fmt.Println("Chat completion error.")
			os.Exit(1)
		}

		res = chatCompletion.Choices[0].Message.Content

	case ".mp3":
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Println("Error opening file...")
			os.Exit(1)
		}
		transcription, err := openAiClient.Audio.Transcriptions.New(
			context.Background(),
			openai.AudioTranscriptionNewParams{
				Model: openai.AudioModelWhisper1,
				File:  file,
			})
		if err != nil {
			fmt.Println("Error during transcription...")
			os.Exit(1)
		}
		res = transcription.Text
	default:
		res = ""
	}

	return res
}

func CategorizeContents(content string, openAiClient openai.Client) string {
	systemMessage := "You are an expert in categorization of notes. You assign the notes to three categories only - people, hardware and other.\n\n'people' - all notes containing information about captured humans or human traces present\n'hardware' - all notes mantioning any hardware bugs and issues\n'other' - everything that does not fit into previous two categories\n\nYou respond only ith the name of the category and nothing else.\n\n<examples>\nNote:\"Janusz został schwytany.\"\nResponse:\"people\"\n\nNote:\"The robot does not have his left eye.\"\nResponse:\"hardware\"\n\nNote:\"Deszcz pada.\"\nResponse:\"other\"\n</examples"
	userMessage := fmt.Sprintf("Which category does this note belong to?\n\n<note>\n %s\n</note>", content)

	chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userMessage),
			openai.SystemMessage(systemMessage),
		},
		Model: openai.ChatModelGPT4o,
	})
	if err != nil {
		fmt.Printf("Cannot categorize with error: %s\n", err)
		os.Exit(1)
	}
	return chatCompletion.Choices[0].Message.Content
}
func SendAnswer(answer Answer, URL string) []byte {
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
