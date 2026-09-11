package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	var obj map[string]map[string]map[string]string
	f,_ := os.Open("../data/messages.json")
	err := json.NewDecoder(f).Decode(&obj)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(obj["str"]["var"])
}