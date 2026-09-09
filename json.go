package main

import (
	"encoding/json"
	"os"
)

func loadMessages() ([]Message, error) {
	mutex.Lock()
	defer mutex.Unlock()

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
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	defer file.Close()

	var accounts map[string]User
	err = json.NewDecoder(file).Decode(&accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil

}

func saveMessages(msgs []Message, messageFile string) error {
	mutex.Lock()
	defer mutex.Unlock()

	file, err := os.Create(messageFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	return encoder.Encode(msgs)
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
