package main

import (
	"encoding/json"
	"fmt"
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

	OpenAiApiKey := os.Getenv("OPENAI_API_KEY")
	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey),
	)

	ApiKey := os.Getenv("API_KEY")
	TaskName := "photos"
	Centrala := os.Getenv("URL_CNTRL")
	ApiUrl := Centrala + "report"
	CacheDir := "cache"

	// Start conversation
	var Message Message
	Message.Apikey = ApiKey
	Message.Task = TaskName
	Message.Answer = "START"

	initialResponse := SendMessage(Message, ApiUrl)

	var urls UrlsJSON
	conv := GetImageURLai(openAiClient, initialResponse.Message, Centrala+"dane/barbara")
	json.Unmarshal([]byte(conv), &urls)

	// Download or load from cache
	photos := make(map[string]string)
	for _, u := range urls.Urls {
		for name, photo := range LoadCachebase64Photo(openAiClient, u, CacheDir) {
			photos[name] = photo
		}
	}

	goodPhotos := make(map[string]string)
	rate := ""
	fn := ""
	ph := ""

	for fileName, photo := range photos {
		fn = fileName
		ph = photo

		for {
			rate = RatePhoto(openAiClient, ph, fn)
			if rate == "SKIP" || strings.Contains(rate, "FOUND") {
				break
			}
			Message.Answer = rate
			Response := SendMessage(Message, ApiUrl)

			var urls UrlsJSON
			conv := GetImageURLai(openAiClient, Response.Message, Centrala+"dane/barbara")
			json.Unmarshal([]byte(conv), &urls)

			if len(urls.Urls) == 0 {
				continue
			}
			fn = FileNameFromURL(urls.Urls[0])

			for name, photo := range LoadCachebase64Photo(openAiClient, urls.Urls[0], CacheDir) {
				fn = name
				ph = photo
			}
		}
		if strings.Contains(rate, "FOUND") {
			goodPhotos[fn] = ph
		}
	}

	description := PhotoDescription(openAiClient, goodPhotos)

	Message.Answer = description
	Response := SendMessage(Message, ApiUrl)
	fmt.Println(Response.Message)
}
