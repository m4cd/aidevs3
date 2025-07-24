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

	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Error while loading .env file.")
	}

	ApiKey := os.Getenv("API_KEY")
	Centrala := os.Getenv("URL_CNTRL")

	var ConnectionsJson ConnectionsJson
	json.Unmarshal([]byte(ReadFileToString("connections.json")), &ConnectionsJson)

	var UsersJson UsersJson
	json.Unmarshal([]byte(ReadFileToString("users.json")), &UsersJson)

	Connections := ConnectionsJson.Reply
	Users := UsersJson.Reply

	NEO4Juri := os.Getenv("NEO4J_URI")
	NEO4Jusername := os.Getenv("NEO4J_USERNAME")
	NEO4Jpassword := os.Getenv("NEO4J_PASSWORD")

	neo4jDriver, err := neo4j.NewDriverWithContext(
		NEO4Juri,
		neo4j.BasicAuth(NEO4Jusername, NEO4Jpassword, ""))
	if err != nil {
		panic(err)
	}
	defer neo4jDriver.Close(context.Background())

	err = neo4jDriver.VerifyConnectivity(context.Background())
	if err != nil {
		panic(err)
	}

	session := neo4jDriver.NewSession(context.Background(), neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})

	// Deleting all users:
	_, err = session.Run(
		context.Background(),
		`MATCH (n:User) DETACH DELETE n;`,
		map[string]any{})
	if err != nil {
		panic(err)
	}

	for _, user := range Users {
		_, err := session.Run(
			context.Background(),
			`CREATE (a:User {
username: $username, 
user_id: $id, 
access_level: $access_level, 
is_active: $is_active, 
last_logon: $last_logon})`,
			map[string]any{
				"username":     user.Username,
				"id":           user.ID,
				"access_level": user.AccessLevel,
				"is_active":    user.IsActive,
				"last_logon":   user.Lastlog,
			})
		if err != nil {
			panic(err)
		}
	}

	for _, conn := range Connections {
		_, err := session.Run(
			context.Background(),
			`MATCH (user1:User {user_id: $user1_id})
MATCH (user2:User {user_id: $user2_id})
CREATE (user1)-[:KNOWS]->(user2)`,
			map[string]any{
				"user1_id": conn.User1,
				"user2_id": conn.User2,
			})
		if err != nil {
			panic(err)
		}
	}

	result, err := session.Run(
		context.Background(),
		`MATCH (source:User {username: 'Rafał'}),
(target:User {username: 'Barbara'}),
p = shortestPath((source)-[:KNOWS*]->(target))
RETURN [n in nodes(p) | n.username] as path`,
		map[string]any{})
	if err != nil {
		panic(err)
	}

	res, _ := result.Single(context.Background())

	names, _ := res.Get("path")
	namesString := fmt.Sprintf("%v", names)
	namesString = strings.Split(namesString, "[")[1]
	namesArray := strings.Fields(strings.Split(namesString, "]")[0])

	answer := strings.Join(namesArray, ",")

	var answ AnswerType
	answ.Answer = answer
	answ.ApiKey = ApiKey
	answ.Task = "connections"

	response := SendAnswer(answ, Centrala+"report")

	fmt.Println(string(response))
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
