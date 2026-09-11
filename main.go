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

type User struct {
	UserName string `json:"userName"`
	PasswordHash string `json:"password"`
	Token string `json:"token"`
	Expires string `json:"expires"`
}

type Message struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Time    string `json:"time"`
}

type Config struct {
	Ip     string  `json:"ip"`
	Port   float64 `json:"port"`
	TLS    bool    `json:"tls"`
	TLSCRT string  `json:"tlsCrt"`
	TLSKEY string  `json:"tlsKey"`
}

const (
	resourseDir       string = "resourse"
	loginFile         string = "html/login.html"
	usersFile      string = "data/users.json"
	indexFile         string = "html/index.html"
	itemFile          string = "html/item.html"
	messageFile       string = "data/messages.json"
	configFile        string = "config.json"
	maxJsonLength     int    = 1024
	maxHttpJsonLength int    = 32
)

func toSha256(data string) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write([]byte(data)); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

var (
	mutex   sync.Mutex
	specURL map[string]func(http.ResponseWriter, *http.Request) = map[string]func(http.ResponseWriter, *http.Request){
		"/":       mainHandler,
		"/login":  loginHandler,
		"/submit": submitHandler,
		"/logout": logoutHandler,
	}
)

func main() {

	config, err := loadConfig()
	if err != nil {
		record.Error(err)
		os.Exit(1)
	}

	var handler handler

	if config.TLS {
		record.Info("https server start on: https://" + config.Ip + ":" + fmt.Sprint(int(config.Port)))
		if err := http.ListenAndServeTLS(config.Ip+":"+fmt.Sprint(int(config.Port)), config.TLSCRT, config.TLSKEY, handler); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		record.Info("http server start on: http://" + config.Ip + ":" + fmt.Sprint(int(config.Port)))
		if err := http.ListenAndServe(config.Ip+":"+fmt.Sprint(int(config.Port)), handler); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
