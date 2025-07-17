package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/qdrant/go-client/qdrant"
)

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error while loading .env file.")
	}

	QdrantApiKey := os.Getenv("QDRANT_API_KEY")
	QdrantHost := os.Getenv("QDRANT_HOST")
	QdrantCollection := "vectors"
	OpenAiApiKey := os.Getenv("OPENAI_API_KEY")
	openAiClient := openai.NewClient(
		option.WithAPIKey(OpenAiApiKey),
	)

	Embedding := openai.EmbeddingModelTextEmbeddingAda002
	const EmbeddingSize = 1536

	FilesDir := "do-not-share"

	ApiKey := os.Getenv("API_KEY")
	Centrala := os.Getenv("URL_CNTRL")

	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host:   QdrantHost,
		Port:   6334,
		APIKey: QdrantApiKey,
		UseTLS: true,
	})

	if err != nil {
		fmt.Println("Cannot create qdrant client.")
		os.Exit(1)
	}

	// create collection if does not exist
	collectionExists, err := qdrantClient.CollectionExists(context.TODO(), QdrantCollection)
	if err != nil {
		fmt.Println("Collection existence check failure.")
	}
	if !collectionExists {
		fmt.Println("Creating new collection...")
		qdrantClient.CreateCollection(context.TODO(), &qdrant.CreateCollection{
			CollectionName: QdrantCollection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     uint64(EmbeddingSize),
				Distance: qdrant.Distance_Cosine,
			}),
		})
	} else {
		fmt.Println("Collection already exists... Continuing...")
	}

	files, err := os.ReadDir(FilesDir)
	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}

	var qdrantPoints []*qdrant.PointStruct
	for _, file := range files {
		dateParts := strings.Split(strings.Split(file.Name(), ".")[0], "_")
		dateFromFilename := dateParts[0] + "-" + dateParts[1] + "-" + dateParts[2]

		fileText := ReadFileToString(FilesDir + "/" + file.Name())
		embeddedFile, err := EmbedString(fileText, openAiClient, Embedding)
		if err != nil {
			fmt.Println("Cannot embed.")
			os.Exit(1)
		}

		float32embedding := ConvertFloat64ArrayToFloat32(embeddedFile.Data[0].Embedding)

		qdrantPoint := qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(uuid.New().String()),
			Vectors: qdrant.NewVectors(float32embedding...),
			Payload: qdrant.NewValueMap(map[string]any{
				"data": dateFromFilename,
				"text": fileText,
			}),
		}

		qdrantPoints = append(qdrantPoints, &qdrantPoint)
	}
	_, err = qdrantClient.Upsert(context.Background(), &qdrant.UpsertPoints{
		CollectionName: QdrantCollection,
		Points:         qdrantPoints,
	})
	if err != nil {
		fmt.Println("Cannot insert points to quadrant.")
	}

	Question := "W raporcie, z którego dnia znajduje się wzmianka o kradzieży prototypu broni?"
	embeddedQuestion, err := EmbedString(Question, openAiClient, Embedding)
	if err != nil {
		fmt.Println("Cannot embed question.")
	}

	fmt.Println("Asking question...")

	var limit uint64 = 1
	embeddedQuestionF32 := ConvertFloat64ArrayToFloat32(embeddedQuestion.Data[0].Embedding)
	qdrantQueryPoints := qdrant.QueryPoints{
		CollectionName: QdrantCollection,
		Query:          qdrant.NewQuery(embeddedQuestionF32...),
		Limit:          &limit,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	}

	qdrantScoredpoints, err := qdrantClient.Query(context.TODO(), &qdrantQueryPoints)
	if err != nil {
		fmt.Println("Cannot query qdrant.")
	}

	answer := qdrantScoredpoints[0].Payload["data"].GetStringValue()

	var ans AnswerType
	ans.ApiKey = ApiKey
	ans.Task = "wektory"
	ans.Answer = answer

	response := SendAnswer(ans, Centrala+"report")
	fmt.Println(string(response))
}

type AnswerType struct {
	Task   string                 `json:"task"`
	ApiKey string                 `json:"apikey"`
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

func ConvertFloat64ArrayToFloat32(f64 []float64) []float32 {
	var f32 []float32
	for _, e := range f64 {
		f32 = append(f32, float32(e))
	}
	return f32
}

func EmbedString(content string, openAiClient openai.Client, embedding openai.EmbeddingModel) (*openai.CreateEmbeddingResponse, error) {
	embd, err := openAiClient.Embeddings.New(context.TODO(), openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(content),
		},
		Model:          embedding,
		EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
		User:           openai.String("aidevs"),
	})
	return embd, err
}

func ReadFileToString(filename string) string {
	b, err := os.ReadFile(filename)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	return string(b)
}
