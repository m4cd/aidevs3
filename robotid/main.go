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

type RobotDescription struct {
	Description string `json:"description"`
}

type Answer struct {
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
	TaskFile := "robotid.json"
	TaskURL := Centrala + "data/" + ApiKey + "/" + TaskFile

	OpenAiApiKey := os.Getenv("OPENAI_API_KEY")
	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey),
	)

	// Downloading task
	err = downloadFile(TaskFile, TaskURL)
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
	var RobotDescription RobotDescription
	json.Unmarshal(b, &RobotDescription)

	description := RobotDescription.Description

	prompt := `Jesteś specjalistą generującym obrazy robotów na podstawie relacji świadków
	
	Opis Robota:
	"` + description + "\""

	params := openai.ImageGenerateParams{}
	params.Model = openai.ImageModelDallE3
	params.Prompt = prompt
	params.N.Value = 1
	params.Size = openai.ImageGenerateParamsSize1024x1024
	params.ResponseFormat = "url"

	imageResponse, err := openAiClient.Images.Generate(
		context.TODO(),
		params,
	)
	if err != nil {
		fmt.Println("Image generation error.")
		os.Exit(1)
	}

	Answer := Answer{}
	Answer.ApiKey = ApiKey
	Answer.Task = "robotid"
	Answer.Answer = imageResponse.Data[0].URL

	resAns := SendAnswer(Answer, Centrala+"report")
	fmt.Println(string(resAns))
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
