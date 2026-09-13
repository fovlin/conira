package admin

import (
	"net/http"
	"os"
	"encoding/json"
	"acovia.net/record"
	"bytes"
)

func AdminIndexHandler(w http.ResponseWriter, r *http.Request) {

	if !validCookie(w, r) {
		login(w)
		return
	}

	indexData, err := os.ReadFile(indexFile)
	if err != nil {
		record.Error("read index file", err)
	}

	itemData, err := os.ReadFile(itemFile)
	if err != nil {
		record.Error("read item file", err)
	}

	messageList, err := loadMessages()
	if err != nil {
		record.Error("load message list: ", err)
	}

	allMessage := []byte{}
	for _, message := range messageList {
		aMessageData := itemData
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ name }}"), []byte(message.Name))
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ time }}"), []byte(message.Time))
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ content }}"), []byte(message.Content))
		allMessage = append(aMessageData, allMessage...)
	}

	if len(allMessage) == 0 {
		allMessage = []byte("<p style=\"text-align: center\">None message</p>")
	}

	nameCookie, err := r.Cookie("UserName")
	if err != nil {
		record.Warn("(load user name from cookie)", err)
		return
	}

	userName := []byte(nameCookie.Value)
	indexData = bytes.ReplaceAll(indexData, []byte("{{ user-name }}"), userName)
	indexData = bytes.ReplaceAll(indexData, []byte("{{ messages }}"), allMessage)
	w.Write(indexData)

}

func validCookie(w http.ResponseWriter, r *http.Request) bool {

	nameCookie, err := r.Cookie("UserName")
	if err != nil {
		return false
	}

	tokenCookie, err := r.Cookie("Token")
	if err != nil {
		return false
	}

	user, err := LoadAdminData(nameCookie.Value)
	if err != nil {
		record.Error("load user data:", err)
		return false
	}

	if user.Token != tokenCookie.Value {
		return false
	}

	return true

}

func loadAdminList() (map[string]User, error) {

	file, err := os.Open(AdminUsersFile)
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

func login(w http.ResponseWriter) {
	loginFile, err := os.ReadFile(loginFile)
	if err != nil {
		record.Error("(read login file)", err)
		os.Exit(1)
	}
	w.Write(loginFile)
}