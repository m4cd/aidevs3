package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

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
	Apidb := Centrala + "apidb"

	Query := QueryType{
		Task:   "database",
		ApiKey: ApiKey,
	}

	sqlContext := "<sql_context>\n"
	var queries []string
	queries = append(queries, "show tables")
	queries = append(queries, "select * from users")
	queries = append(queries, "select * from datacenters")

	for _, q := range queries {
		sqlContext += fmt.Sprintf("Query:\n%s\n\n", q)
		Query.Query = q
		ApiRes := SendQuery(Query, Apidb)
		sqlContext += fmt.Sprintf("Response:\n%v\n\n", string(ApiRes))
	}
	sqlContext += "</sql_context>"

	systemMessage := "You are an SQL expert that returns an SQL query based on the context provided. Context consists of SQL queries and responses in json format that give you needed information. Return only the query in SQL key within json format."

	userMessage := "Jaka jest kwerenda SQL, która zwróci ID wszystkich czynnych datacenter, które są zarządzane przez menadżerów, którzy aktualnie są nieaktywni.\n\n"

	userMessage += sqlContext
	chatCompletion, err := openAiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userMessage),
			openai.SystemMessage(systemMessage),
		},
		Model:       openai.ChatModelGPT4o,
		Temperature: openai.Float(0.0),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{
				Type: constant.JSONObject("json_object"),
			},
		},
	})
	if err != nil {
		panic(err.Error())
	}
	type sql struct {
		SQL string `json:"SQL"`
	}
	LLMquery := chatCompletion.Choices[0].Message.Content
	var LLMqueryJSON sql
	json.Unmarshal([]byte(LLMquery), &LLMqueryJSON)

	Query.Query = LLMqueryJSON.SQL
	ApiRes := SendQuery(Query, Apidb)

	type dcID struct {
		Dcid string `json:"dc_id"`
	}
	type apiReply struct {
		Reply []dcID `json:"reply"`
	}

	var IDs []int64
	var ApiResJSON apiReply
	json.Unmarshal(ApiRes, &ApiResJSON)

	for _, v := range ApiResJSON.Reply {
		i, _ := strconv.ParseInt(v.Dcid, 10, 32)
		IDs = append(IDs, i)
	}

	var finalAnswer AnswerType
	finalAnswer.ApiKey = ApiKey
	finalAnswer.Task = "database"
	finalAnswer.Answer = IDs

	ansResp := SendAnswer(finalAnswer, Centrala+"report")

	fmt.Println(string(ansResp))
}

func ReadFileToString(filename string) string {
	b, err := os.ReadFile(filename)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	return string(b)
}

type QueryType struct {
	Task   string `json:"task"`
	ApiKey string `json:"apikey"`
	Query  string `json:"query"`
}

func SendQuery(query QueryType, URL string) []byte {
	httpClient := http.Client{}

	jsonBytes, err := json.Marshal(query)
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

type AnswerType struct {
	Task   string  `json:"task"`
	ApiKey string  `json:"apikey"`
	Answer []int64 `json:"answer"`
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
