package admin

import (
	"sync"
	"os"
	"encoding/binary"
	"io"
	"encoding/json"
	"fmt"
	"encoding/hex"
)

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

type MessageBlock struct {
	ID   []byte
	Size []byte
	Data []byte
}

type MessageWithID struct {
	ID      string
	Name    string
	Content string
	Time    string
}

const (
	AdminLoginFile         string = "html/admin/login.html"
	AdminUsersFile         string = "data/admin.json"
	AdminIndexFile         string = "html/admin/index.html"
	AdminItemFile          string = "html/admin/item.html"
)

const (
	resourseDir       string = "resourse"
	loginFile         string = "html/login.html"
	usersFile         string = "data/users.json"
	indexFile         string = "html/index.html"
	itemFile          string = "html/item.html"
	messageFile       string = "data/messages.data"
	configFile        string = "config.json"
)

var (
	mutex   sync.Mutex
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

func LoadAdminData(userName string) (User, error) {

	userList, err := loadAdminList()
	if err != nil {
		return User{}, err
	}

	user, ok := userList[userName]
	if !ok {
		return User{}, fmt.Errorf("user %v not found", userName)
	}

	return user, nil

}