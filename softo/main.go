package main

import (
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	"github.com/joho/godotenv"
	"github.com/mendableai/firecrawl-go"
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
	Centrala := os.Getenv("URL_CNTRL")
	FirecrawlApiKey := os.Getenv("FIRECRAWL_API_KEY")
	FirecrawlUrl := "https://api.firecrawl.dev"
	TaskFile := "softo.json"
	SoftoUrl := os.Getenv("SOFTO_URL")
	TaskURL := Centrala + "data/" + ApiKey + "/" + TaskFile

	Questions := GetQuestions(TaskFile, TaskURL)

	app, err := firecrawl.NewFirecrawlApp(FirecrawlApiKey, FirecrawlUrl)
	if err != nil {
		log.Fatalf("Failed to initialize FirecrawlApp: %v", err)
	}

	Answers := make(map[string]interface{})

	for k, question := range Questions {
		fmt.Println("Processing question...")
		fmt.Printf("i = %v, question = %v\n", k, question)

		currentUrl := SoftoUrl
		visitedSites := []string{}
		todoSites := []string{}

		for {
			fmt.Println("Visited sites:")
			fmt.Println(visitedSites)
			fmt.Println("TODO sites:")
			fmt.Println(todoSites)
			fmt.Printf("Processing: %v\n", currentUrl)

			md := scrapeSiteToMarkdown(currentUrl, *app)
			response := AskLLM(openAiClient, question, md, SoftoUrl)

			if response.HasAnswer {
				Answers[k] = response.Answer
				break
			} else {
				for _, nu := range response.NextUrls {
					if !slices.Contains(todoSites, nu) && !slices.Contains(visitedSites, nu) {
						todoSites = append(todoSites, nu)
					}
				}

			}
			if len(todoSites) > 0 {
				if !slices.Contains(visitedSites, todoSites[0]) {
					visitedSites = append(visitedSites, currentUrl)
					currentUrl = todoSites[0]
					todoSites = slices.Delete(todoSites, 0, 1)
				}

			} else {
				break
			}
			fmt.Println("+++++")
			fmt.Println("")
			time.Sleep(7 * time.Second)
		}
		fmt.Println("++++++++++")
		fmt.Println("")
	}

	for i, v := range Answers {
		fmt.Printf("%v: %v\n", i, v)
	}
}
