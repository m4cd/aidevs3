package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"

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

	ApiKey := os.Getenv("API_KEY")
	Centrala := os.Getenv("URL_CNTRL")
	PeopleApi := Centrala + "people"
	PlacesApi := Centrala + "places"

	OpenAiApiKey := os.Getenv("OPENAI_API_KEY")
	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey),
	)
	systemMessage := `Jesteś specjalistą w wyciąganiu informacji z tekstu. Zwracasz dwie listy w formacie json. Jedna zawiera listę imion (i tylko imion; nie zawiera nazwisk) osób wspomnianych w tekście (klucz "people"), a druga nazwy miast (klucz "places"). Listy nie zawierają duplikatów, ani polskich snaków. Obie listy są zapisane dużymi literami.

# przykład
<tekst>
Zosia urodziła się w Poznaniu. Bardzo nie lubi Marcina. Oboje teraz mieszkają w Warszawie.
</tekst>
<odpowiedź>
{
    "people": [
        "ZOFIA",
        "MARCIN"
    ],
    "places": [
        "POZNAN",
        "WARSZAWA"
    ]
}
</odpowiedź>

`

	userMessage := ReadFileToString("barbara.txt")

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
	var PeoplePlacesJSON PeoplePlaces
	json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &PeoplePlacesJSON)

	people := make([]string, 0)
	places := make([]string, 0)
	processedPeople := make([]string, 0)
	processedPlaces := make([]string, 0)

	people = append(people, PeoplePlacesJSON.People...)
	places = append(places, PeoplePlacesJSON.Places...)
	// city := ""
	possibleCities := []string{}

	fmt.Printf("initial people: %v\n", people)
	fmt.Printf("initial places: %v\n", places)
	fmt.Println("============================")

	// for len(people) > 0 && len(places) > 0 {
	for {
		if len(people) > 0 {
			suspect := people[0]
			people = people[1:]
			if !slices.Contains(processedPeople, suspect) {

				fmt.Printf("Suspect: %v\n", suspect)

				res := SendQuery(
					QueryType{
						ApiKey: ApiKey,
						Query:  suspect,
					},
					PeopleApi,
				)
				var resJSON ApiResponse
				json.Unmarshal([]byte(res), &resJSON)

				fmt.Printf("Her/His places: %v\n", resJSON.Message)

				if resJSON.Message != "[**RESTRICTED DATA**]" {
					cities := strings.Split(resJSON.Message, " ")
					for _, c := range cities {
						if slices.Contains(places, c) || slices.Contains(processedPlaces, c) {
							continue
						}
						places = append(places, c)
					}
				}

				processedPeople = append(processedPeople, suspect)
			}
		}

		if len(places) > 0 {
			place := places[0]
			places = places[1:]

			if !slices.Contains(processedPlaces, place) {

				fmt.Printf("Place: %v\n", place)

				res := SendQuery(
					QueryType{
						ApiKey: ApiKey,
						Query:  place,
					},
					PlacesApi,
				)

				var resJSON ApiResponse
				json.Unmarshal([]byte(res), &resJSON)

				fmt.Printf("Its visitors: %v\n", resJSON.Message)

				if resJSON.Message != "[**RESTRICTED DATA**]" {

					persons := strings.Split(resJSON.Message, " ")
					for _, p := range persons {
						pe := strings.Replace(p, "Ł", "L", -1)
						if pe == "BARBARA" {
							fmt.Printf("BARBARA found in %v\n", place)
							possibleCities = append(possibleCities, place)
						}
						if slices.Contains(people, p) || slices.Contains(processedPeople, p) {
							continue
						}

						people = append(people, pe)
					}
				}
				processedPlaces = append(processedPlaces, place)
			}
		}

		fmt.Printf("People queue: %v\n", people)
		fmt.Printf("Places queue: %v\n", places)
		fmt.Println("============================")

		if (len(people) == 0 && len(places) == 0) || people[0] == "GLITCH" {
			break
		}
	}

	fmt.Printf("possibleCities: %v\n", possibleCities)
	fmt.Printf("processedPlaces: %v\n", processedPlaces)

	for _, v := range possibleCities {
		var Answer AnswerType
		Answer.ApiKey = ApiKey
		Answer.Task = "loop"
		Answer.Answer = v
		resAns := SendAnswer(Answer, Centrala+"report")
		var resAnsJSON ApiResponse
		json.Unmarshal(resAns, &resAnsJSON)
		fmt.Println(string(resAns))
	}
}

func ReadFileToString(filename string) string {
	b, err := os.ReadFile(filename)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	return string(b)
}

type PeoplePlaces struct {
	People []string `json:"people"`
	Places []string `json:"places"`
}

type QueryType struct {
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

type ApiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type AnswerType struct {
	Task   string `json:"task"`
	ApiKey string `json:"apikey"`
	Answer string `json:"answer"`
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
