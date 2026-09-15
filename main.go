package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"sync"

	"acovia.net/record"
)

type MessageBlock struct {
	ID   []byte
	Size []byte
	Data []byte
}

type User struct {
	UserName     string `json:"userName"`
	PasswordHash string `json:"password"`
	Token        string `json:"token"`
	Expires      string `json:"expires"`
}

type Message struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Time    string `json:"time"`
}

type MessageWithID struct {
	ID      string
	Name    string
	Content string
	Time    string
}

type Config struct {
	Verify bool      `json:"verify"`
	Ip     string    `json:"ip"`
	Port   float64   `json:"port"`
	TLS    TLSConfig `json:"tls"`
}

type TLSConfig struct {
	Enable bool   `json:"enable"`
	CRT    string `json:"crt"`
	KEY    string `json:"Key"`
}

const (
	resourseDir string = "resourse"
	loginFile   string = "html/login.html"
	loginNoVerifyFile   string = "html/login-no-verify.html"
	usersFile   string = "data/users.json"
	indexFile   string = "html/index.html"
	itemFile    string = "html/item.html"
	messageFile string = "data/messages.data"
	configFile  string = "config.json"
)

func toSha256(data string) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write([]byte(data)); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

var (
	config Config
	mutex   sync.Mutex
	specURL map[string]func(http.ResponseWriter, *http.Request) = map[string]func(http.ResponseWriter, *http.Request){
		"/":       indexHandler,
		"/login":  loginHandler,
		"/submit": submitHandler,
		"/logout": logoutHandler,
	}
)

func main() {

	config, err := loadConfig()
	if err != nil {
		record.Error("...",err)
		os.Exit(1)
	}

	var handler handler

	if config.TLS.Enable {
		record.Info("https server start on: https://" + config.Ip + ":" + fmt.Sprint(int(config.Port)))
		if err := http.ListenAndServeTLS(config.Ip+":"+fmt.Sprint(int(config.Port)), config.TLS.CRT, config.TLS.KEY, handler); err != nil {
			record.Error(err)
			os.Exit(1)
		}
	} else {
		record.Info("http server start on: http://" + config.Ip + ":" + fmt.Sprint(int(config.Port)))
		if err := http.ListenAndServe(config.Ip+":"+fmt.Sprint(int(config.Port)), handler); err != nil {
			record.Error(err)
			os.Exit(1)
		}
	}
}
