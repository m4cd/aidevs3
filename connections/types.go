package main

type ConnectionObj struct {
	User1 string `json:"user1_id"`
	User2 string `json:"user2_id"`
}

type ConnectionsJson struct {
	Reply []ConnectionObj `json:"reply"`
}

type UserObj struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	AccessLevel string `json:"access_level"`
	IsActive    string `json:"is_active"`
	Lastlog     string `json:"lastlog"`
}

type UsersJson struct {
	Reply []UserObj `json:"reply"`
}

type AnswerType struct {
	Task   string `json:"task"`
	ApiKey string `json:"apikey"`
	Answer string `json:"answer"`
}
