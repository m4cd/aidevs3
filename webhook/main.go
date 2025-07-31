package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi"
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
	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey),
	)
	Centrala := os.Getenv("URL_CNTRL")

	serverPort := "8080"

	router := chi.NewRouter()

	router.Handle("/api", ApiHandler(openAiClient, Centrala))

	server := http.Server{
		Handler: middlewareCors(router),
		Addr:    ":" + serverPort,
	}

	server.ListenAndServe()
}

type instructionJSON struct {
	Instruction string `json:"instruction"`
}

func ApiHandler(openAiClient openai.Client, URL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var insJSON instructionJSON

		err := decoder.Decode(&insJSON)
		if err != nil {
			fmt.Printf("Error decoding json from POST request: %v\n", err)
			return
		}

		fmt.Println("+- Instrukcja")
		fmt.Println(insJSON.Instruction)

		fmt.Println("+- Odpowiedź")
		response := AskLLM(openAiClient, insJSON.Instruction)
		fmt.Println(response)

		type AnswerType struct {
			Description string `json:"description"`
		}

		description := AnswerType{
			Description: response,
		}
		respondWithJSON(w, 200, description)
	}
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		next.ServeHTTP(w, r)
	})
}

func AskLLM(openAiClient openai.Client, instrukcja string) string {
	systemMessage := ReadFileToString("prompt.txt") + "\n"

	userMessage := "<instrukcja>\n"
	userMessage += instrukcja + "\n"
	userMessage += "</instrukcja>\n"
	userMessage += "<mapa>\n"
	userMessage += ReadFileToString("mapa.txt") + "\n"
	userMessage += "</mapa>\n"

	chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userMessage),
			openai.SystemMessage(systemMessage),
		},
		Model:       openai.ChatModelGPT4o,
		Temperature: openai.Float(0.0),
	})
	if err != nil {
		panic(err.Error())
	}

	return chatCompletion.Choices[0].Message.Content
}

func ReadFileToString(filename string) string {
	b, err := os.ReadFile(filename)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	return string(b)
}

func respondWithJSON(w http.ResponseWriter, code int, jsonStruct interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	respData, _ := json.Marshal(jsonStruct)
	w.Write(respData)
}
