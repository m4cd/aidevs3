package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	"github.com/openai/openai-go/shared/constant"
)

func SendMessage(msg Message, URL string) Response {
	httpClient := http.Client{}

	payload, err := json.Marshal(msg)
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

func GetImageURLai(openAiClient openai.Client, description string, baseUrl string) string {
	systemMessage := fmt.Sprintf(`Z tekstu przesłanego przez użytkownika wyciągasz adresy do wspomnianych zasobów i składasz je w poprawne URL-e jak w przykładzie. Wynik zwracasz w postaci jsona z kluczem "urls" zawierającym tabelę stringów zawierających linki to wspomnianych zasobów.
Podstawą adresu url jest base_url=%v


<przykład>
# user message:
Lubią poziomki. Zrobiłem 2 zdjęcia: IMG1.jpg i IMG2.png. Ściągniecie je z mojej stronki $base_url/
# response:
{
"urls": [
	"$base_url/IMG1.jpg", 
	"$base_url/IMG2.png"
]
}

# user message:
Lubią grzyby. Zrobiłem zdjęcie: IMG_1234.jpg.
# response:
{
"urls": [
	"$base_url/IMG_1234.jpg"
]
}

</przykład>`, baseUrl)

	userMessage := description

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
	return chatCompletion.Choices[0].Message.Content
}

func RatePhoto(openAiClient openai.Client, imageBase64 string, fileName string) string {
	systemMessage := `Jesteś śledczym specjalizującym się w weryfikacji zdjęć. Oceniasz czy dostarczone zdjęcie o nazwie <NAZWA_PLIKU>. zawiera kobietę i jest na tyle wyraźne, że można je podać obróbce. Na podstawie dostarczonego linku do zdjęcia zwracasz wynik. Opcje jakie masz do wyboru to:
- "REPAIR <NAZWA_PLIKU>" - jeśli zawartość obrazka zawiera dużo szumu, nie przypomina zdjęcia, lub może być uszkodzone,
- "DARKEN <NAZWA_PLIKU>" - jeśli uznasz, że zdjęcie jest prześwietlone i należy je przyciemnić,
- "BRIGHTEN <NAZWA_PLIKU>" - jeśli uznasz, że zdjęcie jest zbyt ciemne i należy je rozjaśnić,
- "SKIP" - zdjęcie nie jest uszkodzone, ale nie zawiera twarzy potrzebnej do sporządzenia rysopisu,
- "FOUND <NAZWA_PLIKU>" - zdjęcie można użyć do sporządzenia rysopisu.`

	userMessage := fileName

	ChatCompletionContentPartImageParam := openai.ChatCompletionContentPartImageParam{
		ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
			URL:    fmt.Sprintf("data:image/jpeg;base64,%v", imageBase64),
			Detail: "high",
		},
		Type: "image_url",
	}

	params := openai.ChatCompletionNewParams{}
	params.Messages = append(params.Messages, openai.SystemMessage(systemMessage))
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
	return chatCompletion.Choices[0].Message.Content
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

func FileNameFromURL(url string) string {
	slice := strings.Split(url, "/")
	return slice[len(slice)-1]
}

func checkFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	return !errors.Is(error, os.ErrNotExist)
}

func LoadCachebase64Photo(openAiClient openai.Client, url string, cacheDir string) map[string]string {
	images := make(map[string]string)

	fileName := FileNameFromURL(url)
	filePath := cacheDir + "/" + fileName

	if !checkFileExists(filePath) {
		err := downloadFile(filePath, url)
		if err != nil {
			panic(err.Error())
		}
	}

	img, err := os.ReadFile(filePath)
	if err != nil {
		panic(err.Error())
	}
	images[fileName] = base64.StdEncoding.EncodeToString(img)

	return images
}

func PhotoDescription(openAiClient openai.Client, imagesBase64 map[string]string) string {
	systemMessage := `Jesteś śledczym specjalizującym się w generowaniu precyzyjnych, dokładnych, szczegółowych rysopisów na podstawie dostarczonych zdjęć. Wspólnym elementem załączonych zdjęć jest to, że znajduje się na nich podejrzana. Twoim zadaniem jest sporządzenie rysopisu owej podejrzenej. Skup się na:
1. Kolorze włosów - podaj ich doskładny odcień,
2. Znakach szczególnych - tatuażach, znamionach itp.,
3. Biżuterii i okularach,
4. Ubraniach.

Bądź bardzo precyzyjny i szczegółowy. Odpowiedz w języku polskim podając jedynie opis bez zbędnych komentarzy.`

	// 	systemMessage := `Twoim zadaniem jest stworzenie szczegółowego opisu postaci widocznej na załączonych ilustracjach.
	// Skup się tylko na postaci pojawiającej się na większości ilustracji.
	// Zwróć uwagę na:
	// 1. Dokładny kolor włosów (bądź bardzo precyzyjny co do odcienia)
	// 2. Cechach charakterystycznych, znaki szczególne, znamiona
	// 3. Okulary i biżuterię
	// 4. Ubranies

	// Bądź bardzo precyzyjny i szczegółowy. Odpowiedz w języku polskim podając jedynie opis bez zbędnych komentarzy.`

	params := openai.ChatCompletionNewParams{}
	params.Messages = append(params.Messages, openai.SystemMessage(systemMessage))
	for _, imgB64 := range imagesBase64 {
		ChatCompletionContentPartImageParam := openai.ChatCompletionContentPartImageParam{
			ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
				URL:    fmt.Sprintf("data:image/jpeg;base64,%v", imgB64),
				Detail: "high",
			},
			Type: "image_url",
		}
		params.Messages = append(params.Messages, openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
			{
				OfImageURL: &ChatCompletionContentPartImageParam,
			},
		}))
	}
	// ChatCompletionContentPartImageParam := openai.ChatCompletionContentPartImageParam{
	// 	ImageURL: openai.ChatCompletionContentPartImageImageURLParam{
	// 		URL:    fmt.Sprintf("data:image/jpeg;base64,%v", imagesBase64),
	// 		Detail: "high",
	// 	},
	// 	Type: "image_url",
	// }

	// params.Messages = append(params.Messages, openai.UserMessage(userMessage))

	params.Model = openai.ChatModelGPT4o

	chatCompletion, err := openAiClient.Chat.Completions.New(
		context.TODO(),
		params,
	)
	if err != nil {
		fmt.Println("Chat completion error.")
		os.Exit(1)
	}
	return chatCompletion.Choices[0].Message.Content
}
