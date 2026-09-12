package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"

	"acovia.net/record"
)

func loadMessages() ([]MessageWithID, error) {

	mutex.Lock()
	defer mutex.Unlock()

	var (
		block       MessageBlock
		messageList []MessageWithID
		message MessageWithID
	)

	block.ID = make([]byte, 8)
	block.Size = make([]byte, 4)

	file, err := os.Open(messageFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	for {

		_, err = file.Read(block.ID)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		_, err = file.Read(block.Size)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		size := binary.LittleEndian.Uint32(block.Size)

		block.Data = make([]byte, size)
		_, err = file.Read(block.Data)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(block.Data, &message); err != nil {
			return nil, err
		}

		message.ID = hex.EncodeToString(block.ID)
		messageList = append(messageList, message)

	}

	return messageList, nil
}

func loadUserList() (map[string]User, error) {

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

func saveUserData(user User) error {

	mutex.Lock()
	defer mutex.Unlock()

	fileData, err := os.ReadFile(usersFile)
	if err != nil {
		return err
	}

	var userMap map[string]User
	
	err = json.Unmarshal(fileData, &userMap)
	if err != nil {
		return err
	}

	userMap[user.UserName] = user

	data, err := json.Marshal(userMap)
	if err != nil {
		return err
	}

	os.WriteFile(usersFile, data, 0611)

	return nil

}

func saveMessages(message Message, messageFile string) error {

	mutex.Lock()
	defer mutex.Unlock()

	var block MessageBlock

	block.ID = make([]byte, 8)
	block.Size = make([]byte, 4)

	if _, err := os.Stat(messageFile); os.IsNotExist(err) {
		record.Warn(messageFile, "not found, create")
		_, err := os.Create(messageFile)
		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(messageFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0611)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := rand.Reader.Read(block.ID); err != nil {
		return err
	}

	block.Data, err = json.Marshal(&message)
	if err != nil {
		return err
	}

	block.Size = binary.LittleEndian.AppendUint32([]byte{}, uint32(len(block.Data)))

	if _, err := file.Write(block.ID); err != nil {
		return err
	}
	if _, err := file.Write(block.Size); err != nil {
		return err
	}
	if _, err := file.Write(block.Data); err != nil {
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
