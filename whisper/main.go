package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type AnswerType struct {
	Task   string `json:"task"`
	ApiKey string `json:"apikey"`
	Answer string `json:"answer"`
}

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error while loading .env file.")
	}
	ApiKey := os.Getenv("API_KEY")
	Centrala := os.Getenv("URL_CNTRL")
	audioDir := "./przesluchania"

	OpenAiApiKey := os.Getenv("OPENAI_API_KEY")
	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey), // defaults to os.LookupEnv("OPENAI_API_KEY")
	)

	commonContext := "Here are transciptions of every recording we obtained.\n\n"

	// Reading audio files one by one
	audioFiles, err := os.ReadDir(audioDir)
	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}
	for _, audioFile := range audioFiles {
		audioFilePath := audioDir + "/" + audioFile.Name()
		fmt.Println("+- Processing file: " + audioFile.Name())

		personName := strings.TrimSuffix(audioFile.Name(), ".m4a")

		commonContext += personName + " said:\n"

		file, err := os.Open(audioFilePath)
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

		commonContext += transcription.Text + "\n\n"

	}

	userMessage := `I need the name of the street where the instute which Andrzej Maj gives lectures in is located. Andrzej Maj is an academic profesor and gives lecture in the Institute.

<context>
`
	systemMessage := `You are an analist specializing in speech transcription analysis for detective purposes. You find what you are asked for precisely and respond only with the name you're asked to find.
	
Analize all the transcription provided in the context one at a time and draw conclusions. 
List all the clues from the transcriptions and all the inconsistencies that you encounter.
Think out loud and explain your reasoning.
Use your inner knowledge about Univeristies and their Institutes.
Finally analize all your conclusions thoroughly and come up with the final answer.
Cross-check the answer you came up with with all the information you have and validate if it actually makes sense.`

	userMessage += commonContext
	userMessage += "</context>"

	// Finding the name
	chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userMessage),
			openai.SystemMessage(systemMessage),
		},
		Model: openai.ChatModelGPT4o,
	})
	if err != nil {
		fmt.Println("Chat completion error...")
		os.Exit(1)
	}
	Answer := chatCompletion.Choices[0].Message.Content
	fmt.Println("## Answer ##")
	fmt.Println(Answer)

	var ans AnswerType
	ans.ApiKey = ApiKey
	ans.Answer = Answer
	ans.Task = "mp3"

	ansResp := SendAnswer(ans, Centrala+"report")
	fmt.Println(string(ansResp))

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
