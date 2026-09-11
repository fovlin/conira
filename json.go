package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"

)

func loadMessages() (map[string]Message, error) {

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

	var msgs map[string]Message
	err = json.NewDecoder(file).Decode(&msgs)
	if err != nil {
		return nil, nil
	}

	return msgs, nil
}

func loadUsers() (map[string]User, error) {

	file, err := os.Open(usersFile)
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

func saveMessages(msg Message, messageFile string) error {

	mutex.Lock()
	defer mutex.Unlock()

	file, err := os.Open(messageFile)
	if err != nil {
		return err
	}

	id := make([]byte, 8)
	if _, err := rand.Reader.Read(id); err != nil {
		return err
	}

	var messages map[string]Message
	err = json.NewDecoder(file).Decode(&messages)
	if err != nil {
		return err
	}

	messages[hex.EncodeToString(id)] = msg

	if err := file.Close(); err != nil {
		return err
	}

	file, err = os.Create(messageFile)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err = encoder.Encode(messages); err != nil {
		return err
	}
	return nil
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
