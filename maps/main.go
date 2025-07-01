package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
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

	imageName := "maps.jpg"
	image, err := os.ReadFile(imageName)
	if err != nil {
		fmt.Println("Error while loading image file.")
		os.Exit(1)
	}
	// os.WriteFile("base64.txt", []byte(base64.StdEncoding.EncodeToString(image)), 0644)

	imageBase64 := base64.StdEncoding.EncodeToString(image)

	userMessage := `Na załączonym obrazku znajduje się 5 wycinków mapy. 4 z nich pochodzą z jednego miasta (jednej okolicy), a 1 fragment nie pasuje do pozostałych. Jako ekspert od nawigacji i tworzenia map:
- zidentyfikuj ulice, charakterystyczne obiekty jak cmentarze, sklepy, szkoły, kościoły i tym podobne oraz układ urbanistyczny.
- zidentyfikuj wszystkie cechy geograficzne jak parki, wzgórza i rzeki
- zwróć uwagę na wszystkie wyróżniające się cechy
	
Na podstawie tego co ustalisz dowiedz się jakie miasto znajduje się na przesłanych 4 wycinkach. Zanim zwrócisz odpowiedź upewnij się, że odpowiedź jest poprawna a Twoje rozumowanie spójne. Zwróć nazwę miasta w formacie zamieszczonym poniżej. Jako odpowiedź podaj tylko i wyłącznie nazwę miasta, które zidentyfikowałeś na mapach.
`

	//userMessage = `ile fragmentów różnych map jest na przesłanym obrazie?`

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
	params.Messages = append(params.Messages, openai.UserMessage("Podpowiedź: W poszukiwanym mieście znajdują się spichlerze i twierdza"))
	params.Model = openai.ChatModelGPT4o
	params.Temperature = param.Opt[float64]{Value: 0.0}

	chatCompletion, err := openAiClient.Chat.Completions.New(
		context.TODO(),
		params,
	)
	if err != nil {
		fmt.Println("Chat completion error.")
		os.Exit(1)
	}

	Answer := chatCompletion.Choices[0].Message.Content
	fmt.Println("## Answer ##")
	fmt.Println(Answer)

}
