package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error while loading .env file.")
	}
	LlmUrl := os.Getenv("LOCAL_LLM")
	ApiKey := os.Getenv("API_KEY")
	Centrala := os.Getenv("URL_CNTRL")
	TaskFile := "cenzura.txt"
	TaskURL := Centrala + "data/" + ApiKey + "/" + TaskFile

	// Downloading task
	err = downloadFile("cenzura.txt", TaskURL)
	if err != nil {
		fmt.Println("Error while downloading task")
		os.Exit(1)
	}

	// Reading contents to variable
	b, err := os.ReadFile(TaskFile) 
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	PromptPrompt := string(b) 

	PromptSystem := `Replace the following parts of the prompt with CENZURA. Do not place any other characters aroung the CENZURA. Always write CENZURA in capital letters. End your response with dot.
For example:
- First and second names ("Jan Nowak" -> "CENZURA")
- Age ("32" -> "CENZURA")
- City ("Wrocław" -> "CENZURA")
- Street and number ("ul. Szeroka 18" -> "ul. CENZURA")

<example>
"Tożsamość podejrzanego: Michał Wiśniewski. Mieszka we Wrocławiu na ul. Słonecznej 20. Wiek: 30 lat."
->
"Tożsamość podejrzanego: CENZURA. Mieszka we CENZURA na ul. CENZURA. Wiek: CENZURA lat."
</examples>
`
	PromptModel := "gemma2:2b"

	var payloadData Payload
	payloadData.Model = PromptModel
	payloadData.System = PromptSystem
	payloadData.Prompt = PromptPrompt
	payloadData.Stream = false

	res := SendMessage(payloadData, LlmUrl)
	fmt.Println("+- Answer")
	fmt.Println(res.Response)

	var ans Answer
	ans.ApiKey = ApiKey
	ans.Answer = res.Response
	ans.Task = "CENZURA"

	ansResp := SendAnswer(ans, Centrala+"report")
	fmt.Println(string(ansResp))
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

func SendMessage(payloadData Payload, URL string) Response {
	httpClient := http.Client{}

	payload, err := json.Marshal(payloadData)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	bodyReader := bytes.NewReader(payload)

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

	var respJson Response
	bodyAnswerBytes, _ := io.ReadAll(res.Body)
	json.Unmarshal(bodyAnswerBytes, &respJson)

	return respJson
}

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
