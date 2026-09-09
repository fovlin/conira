package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"acovia.net/record"
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
	Ip     string  `json:"ip"`
	Port   float64 `json:"port"`
	TLS    bool    `json:"tls"`
	TLSCRT string  `json:"tlsCrt"`
	TLSKEY string  `json:"tlsKey"`
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

func main() {

	config, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/commit", addHandler)
	http.HandleFunc("/logout", logoutHandler)

	if config.TLS {
		record.Info("https server start on: https://" + config.Ip + ":" + fmt.Sprint(int(config.Port)))
		if err := http.ListenAndServeTLS(config.Ip+":"+fmt.Sprint(int(config.Port)), config.TLSCRT, config.TLSKEY, nil); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		record.Info("http server start on: http://" + config.Ip + ":" + fmt.Sprint(int(config.Port)))
		if err := http.ListenAndServe(config.Ip+":"+fmt.Sprint(int(config.Port)), nil); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
