package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/PuerkitoBio/goquery"
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
	ArticleFilename := "arxiv-draft.html"
	ArticleRootUrl := Centrala + "dane/"
	ArticleUrl := ArticleRootUrl + ArticleFilename
	QuestionsUrl := Centrala + "data/" + ApiKey + "/arxiv.txt"
	ImagesDir := "images"
	Mp3Dir := "mp3"

	// Downloading Questions
	Questions := DownloadToString(QuestionsUrl)

	// Downloading article
	httpClient := http.Client{}
	articleResponse, err := httpClient.Get(ArticleUrl)
	if err != nil {
		fmt.Printf("Cannot get article: %s\n", err)
		os.Exit(1)
	}
	defer articleResponse.Body.Close()

	// Parsing for images and mp3
	Article, err := goquery.NewDocumentFromReader(articleResponse.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Images
	_ = Article.Find("figure").Each(func(index int, figure *goquery.Selection) {
		img := figure.Find("img")
		src, _ := img.Attr("src")

		figcaption := figure.Find("figcaption")
		caption := figcaption.Text()

		response, err := http.Get(ArticleRootUrl + src)
		if err != nil {
			fmt.Printf("Cannot parse images from the article: %s\n", err)
			os.Exit(1)
		}
		defer response.Body.Close()

		f, _ := os.Create(ImagesDir + "/" + src[2:])
		defer f.Close()
		io.Copy(f, response.Body)

		err = os.WriteFile("cache/"+src[2:]+".txt", []byte(caption), 0666)
		if err != nil {
			log.Fatal(err)
		}
	})

	//MP3
	_ = Article.Find("audio").Each(func(index int, item *goquery.Selection) {
		source := item.Find("source")
		src, _ := source.Attr("src")

		response, err := http.Get(ArticleRootUrl + src)
		if err != nil {
			fmt.Printf("Cannot parse images from the article: %s\n", err)
			os.Exit(1)
		}
		defer response.Body.Close()

		f, _ := os.Create(Mp3Dir + "/" + src[2:])
		defer f.Close()
		io.Copy(f, response.Body)
	})

	// Prepare and cache md article
	ArticleCached := "cache/" + ArticleFilename
	_ = downloadFile(ArticleCached, ArticleUrl)

	ArticleHtml := ReadFileToString(ArticleCached)
	ArticleMd, err := htmltomarkdown.ConvertString(ArticleHtml)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(ArticleMd)

	ArticleMdFilepath := "cache/article.md"
	err = os.WriteFile(ArticleMdFilepath, []byte(ArticleMd), 0666)
	if err != nil {
		log.Fatal(err)
	}

	// Photo description generation
	imgFiles, err := os.ReadDir(ImagesDir)

	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}

	for _, file := range imgFiles {
		imagePath := ImagesDir + "/" + file.Name()
		image, err := os.ReadFile(imagePath)

		if err != nil {
			fmt.Println("Error while loading image file.")
			os.Exit(1)
		}
		imageBase64 := base64.StdEncoding.EncodeToString(image)

		userMessage := `Krótko, ale zwięźle opisujesz przesłane obrazy. Nie podajesz zbędnych szczegółów, jak np. to czy obraz jest kolorowy czy w odcieniach szarości. Opisujesz tylko istotne rzeczy, widoczne na pierwszym planie. Wykonaj zadanie w 2 zdaniach max.`
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
		params.Model = openai.ChatModelGPT4o

		chatCompletion, err := openAiClient.Chat.Completions.New(
			context.TODO(),
			params,
		)
		if err != nil {
			fmt.Println("Chat completion error.")
			os.Exit(1)
		}

		res := chatCompletion.Choices[0].Message.Content

		_ = os.WriteFile("cache/"+file.Name()+".ai"+".txt", []byte(res), 0666)
		if err != nil {
			log.Fatal(err)
		}

	}

	// mp3 transcription
	mp3Files, err := os.ReadDir(Mp3Dir)

	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}

	for _, file := range mp3Files {
		mp3Path := Mp3Dir + "/" + file.Name()
		mp3, err := os.Open(mp3Path)
		if err != nil {
			fmt.Println("Error while loading mp3 file.")
			os.Exit(1)
		}
		transcription, err := openAiClient.Audio.Transcriptions.New(
			context.Background(),
			openai.AudioTranscriptionNewParams{
				Model: openai.AudioModelWhisper1,
				File:  mp3,
			})
		if err != nil {
			fmt.Println("Error during transcription...")
			os.Exit(1)
		}
		_ = os.WriteFile("cache/"+file.Name()+".ai"+".txt", []byte(transcription.Text), 0666)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Combining main prompt
	imagesDescriptions := ""
	imgFiles, err = os.ReadDir(ImagesDir)

	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}

	for _, file := range imgFiles {
		imagesDescriptions += file.Name() + "\n"

		imagesDescriptions += "- " + ReadFileToString("cache/"+file.Name()+".txt") + "\n"
		imagesDescriptions += "- " + ReadFileToString("cache/"+file.Name()+".ai.txt") + "\n\n"
	}

	mp3Descriptions := ""
	mp3Files, err = os.ReadDir(Mp3Dir)

	if err != nil {
		fmt.Println("Error while os.ReadDir() ...")
	}

	for _, file := range mp3Files {
		mp3Descriptions += file.Name() + "\n"

		mp3Descriptions += "- \"" + ReadFileToString("cache/"+file.Name()+".ai.txt") + "\"\n\n"
	}

	userMessage := "Pytania:\n" + Questions + "\n" + "Kontekst:\n"
	userMessage += "+- TEKST\n" + ArticleMd + "\n\n"
	userMessage += "+- OBRAZKI\n" + imagesDescriptions
	userMessage += "+- AUDIO\n" + mp3Descriptions

	systemMessage := ReadFileToString("system_prompt.txt")

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
	Answer := chatCompletion.Choices[0].Message.Content
	var ans AnswerType
	ans.ApiKey = ApiKey
	ans.Task = "arxiv"

	var jsonAnswer map[string]interface{}
	json.Unmarshal([]byte(Answer), &jsonAnswer)
	ans.Answer = jsonAnswer

	response := SendAnswer(ans, Centrala+"report")
	fmt.Println(string(response))
}

func ReadFileToString(filename string) string {
	b, err := os.ReadFile(filename)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	return string(b)
}

func DownloadToString(url string) string {
	tempFile := "temp.txt"
	err := downloadFile(tempFile, url)
	if err != nil {
		fmt.Println("Error while downloading task")
		os.Exit(1)
	}

	// Reading contents to variable
	fileContents := ReadFileToString(tempFile)

	err = os.Remove(tempFile)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	return string(fileContents)
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

type AnswerType struct {
	Task   string                 `json:"task"`
	ApiKey string                 `json:"apikey"`
	Answer map[string]interface{} `json:"answer"`
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
