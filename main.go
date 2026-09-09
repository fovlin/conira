package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type User struct {
	UserName string
	Password string
}

type Message struct {
	Name    string
	Content string
	Time    string
}

type Config struct {
	Ip                string
	Port              float64
	TLS               bool
	TLS_CRT           string
	TLS_KEY           string
}

const (
	loginFile         = "login.html"
	accountsFile      = "users.json"
	indexFile         = "index.html"
	itemFile          = "item.html"
	messageFile       = "messages.json"
	configFile        = "config.json"
	maxJsonLength     = 1024
	maxHttpJsonLength = 32
)

var (
	mutex sync.Mutex
)

func loadMessages() ([]Message, error) {
	file, err := os.Open(messageFile)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var msgs []Message
	err = json.NewDecoder(file).Decode(&msgs)
	if err != nil {
		return nil, nil
	}
	return msgs, nil
}

func loadUser() (map[string]User, error) {

	file, err := os.Open(accountsFile)
	if os.IsNotExist(err) {
		fmt.Println(err)
		return nil, nil
	}
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	defer file.Close()

	var accounts map[string]User
	err = json.NewDecoder(file).Decode(&accounts)
	if err != nil {
		return nil, nil
	}
	return accounts, nil

}

func saveMessages(msgs []Message, messageFile string) error {
	file, err := os.Create(messageFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	return encoder.Encode(msgs)
}

func login(w http.ResponseWriter) {
	loginFile, err := os.ReadFile(loginFile)
	if err != nil {
		fmt.Println("404 Not Found: " + indexFile)
		os.Exit(1)
	}
	w.Write(loginFile)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	userName := r.FormValue("user-name")
	password := r.FormValue("password")
	if userName == "" || password == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	userList, err := loadUser()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	user, ok := userList[userName]

	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if user.Password != password {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	cookie := &http.Cookie{
		Name:    "IsLogin",
		Value:   user.UserName,
		Secure:  true,
		Expires: time.Now().Add(time.Hour * 24 * 3),
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func mainHandler(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("IsLogin")
	if err == http.ErrNoCookie {
		login(w)
		return
	} else if err != nil {
		fmt.Println(err)
	}

	indexData, err := os.ReadFile(indexFile)
	if err != nil {
		fmt.Println("404 Not Found: " + indexFile)
	}

	itemData, err := os.ReadFile(itemFile)
	if err != nil {
		fmt.Println("404 Not Found: " + itemFile)
	}

	messageList, err := loadMessages()

	if len(messageList) > maxHttpJsonLength {
		messageList = messageList[:maxHttpJsonLength]
	}

	allMessage := []byte{}

	for _, Message := range messageList {
		aMessageData := itemData
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ name }}"), []byte(Message.Name))
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ time }}"), []byte(Message.Time))
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ content }}"), []byte(Message.Content))
		allMessage = append(allMessage, aMessageData...)
	}

	if len(allMessage) == 0 {
		allMessage = []byte("<p style=\"margin: auto auto\">None any message</p>")
	}

	userName := []byte(cookie.Value)
	indexData = bytes.ReplaceAll(indexData, []byte("{{ user-name }}"), userName)
	indexData = bytes.ReplaceAll(indexData, []byte("{{ messages }}"), allMessage)
	w.Write(indexData)

}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	cookie, err := r.Cookie("IsLogin")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	name := cookie.Value

	content := r.FormValue("content")
	if content == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	msgs, _ := loadMessages()
	msg := Message{
		Name:    name,
		Content: content,
		Time:    time.Now().Format(time.DateTime),
	}
	msgs = append([]Message{msg}, msgs...)

	if len(msgs) > maxJsonLength {
		msgs = msgs[:maxJsonLength]
	}

	saveMessages(msgs, messageFile)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {

	config, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/commit", addHandler)

	if config.TLS {
		fmt.Println("server start on: https://" + config.Ip + ":" + fmt.Sprint(int(config.Port)))
		if err := http.ListenAndServeTLS(config.Ip+":"+fmt.Sprint(int(config.Port)), config.TLS_CRT, config.TLS_KEY, nil); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		fmt.Println("server start on: http://" + config.Ip + ":" + fmt.Sprint(int(config.Port)))
		if err := http.ListenAndServe(config.Ip+":"+fmt.Sprint(int(config.Port)), nil); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func loadConfig() (Config, error) {

	var config Config

	configFile, err := os.Open(configFile)
	if err != nil {
		return Config{}, err
	}
	defer configFile.Close()

	err = json.NewDecoder(configFile).Decode(&config)
	if err != nil {
		return Config{}, err
	}

	return config, nil

}
