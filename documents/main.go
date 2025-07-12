package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/openai/openai-go/shared/constant"
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

	ApiKey := os.Getenv("API_KEY")
	Centrala := os.Getenv("URL_CNTRL")

	FilesDir := "pliki"
	FactsDir := FilesDir + "/" + "facts"
	CacheDir := "cache"

	// Going through txt facts
	// Process facts only once (when there is no info cached)
	cache, err := os.ReadDir(CacheDir)
	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}
	if len(cache) == 0 {
		facts, err := os.ReadDir(FactsDir)
		if err != nil {
			fmt.Println("Error while os.ReadDir() ...")
		}
		for _, factFile := range facts {
			switch path.Ext(factFile.Name()) {
			case ".txt":
				userMessage := ReadFileToString(FactsDir + "/" + factFile.Name())

				systemMessage := `Jesteś śledczym analizującym ekstrahującym kluczowe informacje z dostarczonych faktów w postaci opisów. Z dostarczonego opisu wypisz kluczowe informacje (np. osoby, ich zawody, specjalne umiejętności, co się stało z tymi osobmi, gdzie, miejsca o których mowa, jakie przedmioty i technologie były użyte). Weź pod uwagę to, że nazwiskach mogą być literówki (np. "Kowaski" i "Kowalki"). Rozpoznaj z kotekstu, kiedy mowa o tej samej osobie.`

				ChatResponse := CompleteChat(openAiClient, userMessage, systemMessage)
				_ = os.WriteFile("cache"+"/"+factFile.Name(), []byte(ChatResponse), 0666)

			}
		}
	}

	// Going through txt reports
	ReportsPrimaryConclusions := ""
	files, err := os.ReadDir(FilesDir)
	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}
	for _, file := range files {
		switch path.Ext(file.Name()) {
		case ".txt":
			userMessage := "Nazwa pliku:\n" + file.Name() + "\n"
			userMessage += "Zawartość pliku:\n"
			userMessage += ReadFileToString(FilesDir + "/" + file.Name())

			systemMessage := `Jesteś śledczym analizującym rapotry i wydobywającym słowa kluczowe. Nie rozpisuj się pełnymi zdaniami. Zidentyfikuj kluczowe informacje: co się stało, gdzie, kto był zaangażowany, jakie przedmioty/technologie się pojawiły. Analizując weź pod uwagę również nazwę pliku, bo tam też mogą być ważne poszlaki. Odpowiedź zwróć w postaci listy słów kluczowych, oddzielonych przecinkami.`

			ChatResponse := CompleteChat(openAiClient, userMessage, systemMessage)
			ReportsPrimaryConclusions += "Nazwa pliku:\n" + file.Name() + "\n" + "Słowa kluczowe:\n" + ChatResponse + "\n==========================================\n"
		}
	}

	userMessage := "<fakty>\n"

	facts, err := os.ReadDir(CacheDir)
	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}
	for _, factFile := range facts {
		userMessage += ReadFileToString(CacheDir + "/" + factFile.Name())
		userMessage += "/n==========================================/n"
	}
	userMessage += "</fakty>\n"
	userMessage += "<raporty>\n"
	files, err = os.ReadDir(FilesDir)
	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}
	for _, report := range files {
		switch path.Ext(report.Name()) {
		case ".txt":
			userMessage += "Nazwa raportu: " + report.Name() + "\n"
			userMessage += ReadFileToString(FilesDir + "/" + report.Name())
			userMessage += "\n==========================================\n"
		}
	}
	userMessage += "</raporty>\n"
	systemMessage := `Jesteś śledczym łączącym w logiczny i spójny sposób informacje wyekstrahowane z faktów (tag <fakty>) z dostarczonymi kluczowymi słowami pochodzącymi z raportów (tag <raporty>). Zwróć odpowiedź w postaci json-a, gdzie kluczem jest nazwa pliku z raportem, a wartością string w postaci listy słów kluczowych, które uznasz za najistotniejsze oddzielonych przecinkami.

Uwzględnij kontekst w postaci faktów o osobie, jeśli zostały podane, takich jak jej zawód (bardzo konkretnie), jak się nazywa, gdzie zamieszkuje, jakie specjalne umiejętności posiada, na jakich technologiach się zna (jęzki programowania), co się z nią stało, gdzie, jakie przedmioty i technologie były użyte. 

Zwróć odpowiedź w postaci json-a, gdzie kluczem jest nazwa pliku z raportem, a wartością string w postaci listy minimum 20 słów kluczowych, które uznasz za najistotniejsze oddzielonych przecinkami.`

	ChatResponse := CompleteChatJson(openAiClient, userMessage, systemMessage)
	fmt.Println("### RESPONSE ###")
	fmt.Println(ChatResponse)

	var ans AnswerType
	ans.ApiKey = ApiKey
	ans.Task = "dokumenty"

	var jsonAnswer map[string]interface{}
	json.Unmarshal([]byte(ChatResponse), &jsonAnswer)
	ans.Answer = jsonAnswer

	response := SendAnswer(ans, Centrala+"report")
	fmt.Println(string(response))
}

func ReadFileToString(filename string) string {
	b, err := os.ReadFile(filename)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	return string(b)
}

func CompleteChat(openAiClient openai.Client, userMessage string, systemMessage string) string {
	chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userMessage),
			openai.SystemMessage(systemMessage),
		},
		Model:       openai.ChatModelGPT4o,
		Temperature: openai.Float(0.1),
	})

	if err != nil {
		panic(err.Error())
	}

	return chatCompletion.Choices[0].Message.Content
}

func CompleteChatJson(openAiClient openai.Client, userMessage string, systemMessage string) string {
	chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userMessage),
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
