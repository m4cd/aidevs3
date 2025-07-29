package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/mendableai/firecrawl-go"
	"github.com/openai/openai-go"
)

func downloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func ReadFileToString(filename string) string {
	b, err := os.ReadFile(filename)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	return string(b)
}

func scrapeSiteToMarkdown(url string, app firecrawl.FirecrawlApp) string {
	scrapeStatus, err := app.ScrapeURL(url, &firecrawl.ScrapeParams{
		Formats: []string{"markdown", "html"},
	})
	if err != nil {
		log.Fatalf("Failed to send scrape request: %v", err)
	}

	scrapeMarkdown, err := htmltomarkdown.ConvertString(scrapeStatus.HTML)
	if err != nil {
		log.Fatal(err)
	}

	return scrapeMarkdown
}

func AskLLM(openAiClient openai.Client, question string, md string, baseUrl string) LlmResponseType {
	userMessage := "<question>\n"
	userMessage += question + "/n"
	userMessage += "</question>\n<website>\n"
	userMessage += md + "/n"
	userMessage += "</website>\n"
	systemMessage := fmt.Sprintf(`Jesteś ekspertem w analizie stron internetowych przedstawionych w formacie Markdown. Twoim zadaniem jest odpowiedzieć, jak najbardziej precyzyjnie, na zadane przez użytkownika pytanie (bez opisów lub komentarzy) zawarte w tagu <pytanie> na podstawie strony intenetowej w formacie markdown w tagu <website>.
W przypadku, gdy na danej stronie nie znajdziesz odpowiedzi na postawione pytania wskaż adres podstrony (lub kilku podstron) na które warto zajrzeć w następnej kolejności, żeby znaleźć odpowiedź na dane pytanie. Podstrony muszą być częścią bazowego adresu %v. Pamiętaj, że na blogu korporacji też można znaleźć wartościowe informacje.

Wymaganie względem odpowiedzi:
- rozumowanie: opisz swoje rozumowanie dotyczące wyciągniętych przez Ciebie wniosków
- znaleziono: wartość logiczna "true" oznacza, że znalazłeś ostateczną odpowiedź (wartość "false", że nie)
- odpowiedź: precyzyjna odpowiedź na zadanie pytanie (jeśli taką znaleziono, w przeciwnym wypadku zwróć null)
- lista_url: lista odnośników na które warto zajrzeć w następnej kolejności

przykładowa odpowiedź:
{
    "thinking": "Strona (https://janusz.pl/about) jest na temat taki, lecz nie zawiera nawet wzmianki na to co jest zawarte w pytaniu. Kontekst pytania sugeruje, że warto zajrzeć do zakładek 'details' oraz 'advanced'. Pozostałe zakładki zdają się być niepowiązanie z pytaniem.",
    "has_answer": false,
    "answer": null,
    "next_urls": ["https://janusz.pl/details", "https://janusz.pl/advanced", "https://janusz.pl/technical-specs"]
}`, baseUrl)

	chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userMessage),
			openai.SystemMessage(systemMessage),
		},
		Model:       openai.ChatModelGPT4_1,
		Temperature: openai.Float(0.0),
		// ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
		// 	OfJSONObject: &shared.ResponseFormatJSONObjectParam{
		// 		Type: constant.JSONObject("json_object"),
		// 	},
		// },
	})
	if err != nil {
		panic(err.Error())
	}

	var res LlmResponseType
	json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &res)
	return res
}

func GetQuestions(TaskFile string, TaskURL string) map[string]string {
	err := downloadFile(TaskFile, TaskURL)
	if err != nil {
		fmt.Println("Error while downloading task")
		os.Exit(1)
	}

	Tasks := ReadFileToString(TaskFile)

	var Questions map[string]string
	json.Unmarshal([]byte(Tasks), &Questions)

	return Questions
}

func (response LlmResponseType) Print() {
	fmt.Printf("thinking: %v\n", response.Thinking)
	fmt.Printf("has_answer: %v\n", response.HasAnswer)
	fmt.Printf("answer: %v\n", response.Answer)
	fmt.Printf("next_urls: %v\n", response.NextUrls)
}
