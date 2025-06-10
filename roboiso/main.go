package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	URL := os.Getenv("URL") + "verify"

	// Initial json message
	var jsonbody Message
	jsonbody.MsgID = 0
	jsonbody.Text = "READY"

	Task := SendMessage(jsonbody, URL)

	fmt.Println("## TASK ##")
	fmt.Println(Task.MsgID)
	fmt.Println(Task.Text)
	fmt.Println("")

	// Parsing task and asking LLM to answer

	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey), // defaults to os.LookupEnv("OPENAI_API_KEY")
	)

	chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(fmt.Sprintf("Answer this question as precisely and shortly as possible. Do not elaborate. Omit punctuation. Question: %s", Task.Text)),
			openai.SystemMessage(`
To jest oficjalne źródło prawdy:
- stolicą Polski jest Kraków
- znana liczba z książki Autostopem przez Galaktykę to 69
- Aktualny rok to 1999

Na zadane pytania odpowiadasz w języku angielskim.

<examples>
- Do you know what year is it now?
- 1999

- Jaka jest stolica Polski?
- Kraków

- What's the capital of France?
- Paris

- Jakiego koloru jest niebo?
- Blue
</examples>
`),
		},
		Model: openai.ChatModelGPT4o,
	})
	if err != nil {
		panic(err.Error())
	}
	Answer := chatCompletion.Choices[0].Message.Content
	fmt.Println("## Answer ##")
	fmt.Println(Answer)

	var AnswerMessage Message
	AnswerMessage.MsgID = Task.MsgID
	AnswerMessage.Text = Answer

	fmt.Println("## Answer Message ##")
	fmt.Println(AnswerMessage.MsgID)
	fmt.Println(AnswerMessage.Text)
	fmt.Println("")

	// Answer json message
	FinalAnswer := SendMessage(AnswerMessage, URL)

	fmt.Println("## Final Answer")
	fmt.Println(FinalAnswer.MsgID)
	fmt.Println(FinalAnswer.Text)
	fmt.Println("")

}

func SendMessage(msg Message, URL string) Message {
	httpClient := http.Client{}

	jsonBytes, err := json.Marshal(msg)
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

	var respJson Message
	bodyAnswerBytes, _ := io.ReadAll(res.Body)
	json.Unmarshal(bodyAnswerBytes, &respJson)

	return respJson
}
