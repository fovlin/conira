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
    Ip string
    Port float64
    TLS bool
    TLS_CRT string
    TLS_KEY string
}

var (
    accountsFile = "accounts.json"
    indexFile = "index.html"
    itemFile = "item.html"
    messageFile = "messages.json"
    configFile = "config.json"
    mutex    sync.Mutex
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
        return nil, nil
    }
    if err != nil {
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

func saveMessages(msgs []Message) error {
    file, err := os.Create(messageFile)
    if err != nil {
        return err
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "    ")
    return encoder.Encode(msgs)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    indexData, err := os.ReadFile(indexFile)
    if err != nil {
        fmt.Println("404 Not Found: " + indexFile)
        os.Exit(1)
    }

    itemData, err := os.ReadFile(itemFile)
    if err != nil {
        fmt.Println("404 Not Found: " + itemFile)
        os.Exit(1)
    }

    MessageList, err := loadMessages()
    reSortMessageList := make([]Message, len(MessageList))
    allMessage := []byte{}

    for i := 0; i <= len(MessageList) - 1; i++ {
        reSortMessageList[i] = MessageList[len(MessageList) - i - 1]
    }

    for _, Message := range reSortMessageList {
        aMessageData := itemData
        aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ name }}"), []byte(Message.Name))
        aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ time }}"), []byte(Message.Time))
        aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ content }}"), []byte(Message.Content))
        allMessage = append(allMessage, aMessageData...)
    }

    if len(allMessage) == 0 {
        allMessage = []byte("<p style=\"margin: auto auto\">None any message</p>")
    }

    indexData = bytes.ReplaceAll(indexData, []byte("{{ messages }}"), allMessage)
    w.Write(indexData)

}

func addHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }

    name := r.FormValue("name")
    content := r.FormValue("content")
    if name == "" || content == "" {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }

    mutex.Lock()
    defer mutex.Unlock()

    msgs, _ := loadMessages()
    msgs = append(msgs, Message{
        Name:    name,
        Content: content,
        Time:    time.Now().Format(time.DateTime),
    })

    saveMessages(msgs)

    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {

    var config Config

    configFile, err := os.Open(configFile)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    defer configFile.Close()

    err = json.NewDecoder(configFile).Decode(&config)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    
    http.HandleFunc("/", indexHandler)
    http.HandleFunc("/add-note", addHandler)

    if config.TLS {
        fmt.Println("server start on: https://" + config.Ip + ":" + fmt.Sprint(int(config.Port)))
        if err := http.ListenAndServeTLS(config.Ip + ":" + fmt.Sprint(int(config.Port)), config.TLS_CRT, config.TLS_KEY, nil); err != nil {
            fmt.Println(err)
            os.Exit(1)
        } else {
        }
    } else {
        fmt.Println("server start on: http://" + config.Ip + ":" + fmt.Sprint(int(config.Port)))
        if err := http.ListenAndServe(config.Ip + ":" + fmt.Sprint(int(config.Port)), nil); err != nil {
            fmt.Println(err)
            os.Exit(1)
        } else {
        }
    }
}