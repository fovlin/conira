package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"acovia.net/record"
)

type handler struct{}

func (handler handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	record.Info("receive request: url:", r.RequestURI, "adddress:", r.RemoteAddr)
	if f, ok := specURL[r.RequestURI]; ok {
		f(w, r)
		return
	}

	resourseRoot, err := os.OpenRoot(resourseDir)
	if err != nil {
		record.Error("(open resourse directory)", err)
	}
	defer resourseRoot.Close()

	filePath, _ := strings.CutPrefix(r.RequestURI, "/")
	file, err := resourseRoot.Open(filePath)
	if err != nil {
		record.Warn("(open resourse file)", err)
	}
	defer file.Close()

	setMime(filePath, w)
	io.Copy(w, file)

}

func login(w http.ResponseWriter) {
	loginFile, err := os.ReadFile(loginFile)
	if err != nil {
		record.Error("(read login file)", err)
		os.Exit(1)
	}
	w.Write(loginFile)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	httpUserName := r.FormValue("user-name")
	httpPassword := r.FormValue("password")
	if httpUserName == "" || httpPassword == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user, err := getUserData(httpUserName)
	if err != nil {
		record.Error("(load user data)", err)
		return
	}

	expiresTime, err := time.Parse(time.DateTime, user.Expires)
	if err != nil {
		record.Warn("(parse user expires time)", err)
	}

	token := make([]byte, 16)
	token, err = hex.DecodeString(user.Token)
	if err != nil {
		record.Error("(load user list)", err)
		return
	}

	if time.Now().After(expiresTime) {

		expiresTime = time.Now().Add(time.Hour * 24 * 3)
		rand.Reader.Read(token)

	}

	httpPasswdHash, err := toSha256(httpPassword)
	record.Debug(httpPasswdHash)

	if user.PasswordHash != httpPasswdHash {
		record.Warn("password verification failed:", "user:", httpUserName)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	nameCookie := &http.Cookie{
		Name:    "UserName",
		Value:   user.UserName,
		Secure:  true,
		Expires: time.Now().Add(time.Hour * 24 * 3),
	}

	tokenCookie := &http.Cookie{
		Name:    "Token",
		Value:   hex.EncodeToString(token),
		Secure:  true,
		Expires: time.Now().Add(time.Hour * 24 * 3),
	}

	http.SetCookie(w, tokenCookie)
	http.SetCookie(w, nameCookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)

	if err = saveUserData(user); err != nil {
		record.Error("save user data", err)
	}

}

func mainHandler(w http.ResponseWriter, r *http.Request) {

	if !checkCookie(w, r) {
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

	// if len(messageList) > maxHttpJsonLength {
	// 	messageList = messageList[:maxHttpJsonLength]
	// }


	var sortedMessages []map[string]Message

	for id, message := range messageList {
		for n, sortedMessageObj := range sortedMessages {
			sortedMessage := sortedMessageObj[id]
			if len(sortedMessage.Time) == 0 {
				sortedMessages[n] = sortedMessageObj
				break
			}

			unsortTime, err := time.Parse(time.DateTime, message.Time)
			if err != nil {
				record.Error("parse time:", err)
			}

			sortedTime, err := time.Parse(time.DateTime, sortedMessage.Time)
			if err != nil {
				record.Error("parse time:", err)
			}

			if unsortTime.Before(sortedTime) {
				prefixList := sortedMessages[:n]
				suffixList := sortedMessages[n:]
				sortedMessages = append(prefixList, sortedMessageObj)
				sortedMessages = append(sortedMessages, suffixList...)
			}
		}
	}

	allMessage := []byte{}
	for _, message := range messageList {
		aMessageData := itemData
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ name }}"), []byte(message.Name))
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ time }}"), []byte(message.Time))
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ content }}"), []byte(message.Content))
		allMessage = append(allMessage, aMessageData...)
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

func submitHandler(w http.ResponseWriter, r *http.Request) {

	if !checkCookie(w, r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}


	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	nameCookie, err := r.Cookie("UserName")
	if err != nil {
		record.Warn("load user name from cookie:", err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	userName := nameCookie.Value

	content := r.FormValue("content")
	if content == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	msg := Message{
		Name:    userName,
		Content: content,
		Time:    time.Now().Format(time.DateTime),
	}

	// if len(msgs) > maxJsonLength {
	// 	msgs = msgs[:maxJsonLength]
	// }

	err = saveMessages(msg, messageFile)
	if err != nil {
		record.Error("save message:",err)
	}

	record.Info("new message:", "user:", msg.Name, "address:", r.RemoteAddr)

	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	nameCookie, err := r.Cookie("UserName")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		record.Error("(parse cookie)", err)
	}
	tokenCookie, err := r.Cookie("UserName")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		record.Error("(parse cookie)", err)
	}

	nameCookie.MaxAge = -1
	tokenCookie.MaxAge = -1

	http.SetCookie(w, nameCookie)
	http.SetCookie(w, tokenCookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func setMime(filePath string, w http.ResponseWriter) {
	switch path.Ext(filePath) {
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	}
}

func checkCookie(w http.ResponseWriter, r *http.Request) bool {

	nameCookie, err := r.Cookie("UserName")
	if err != nil {
		record.Warn("(check cookie)", err)
		return false
	}

	tokenCookie, err := r.Cookie("Token")
	if err != nil {
		record.Warn("(check cookie)", err)
		return false
	}

	user, err := getUserData(nameCookie.Value)
	if err != nil {
		record.Error("get user data", err)
		return false
	}

	if user.Token != tokenCookie.Value {
		return false
	}

	return true

}

func saveUserData(user User) error {

	mutex.Lock()
	defer mutex.Unlock()

	file, err := os.Open(usersFile)
	if err != nil {
		return err
	}
	json.NewEncoder(file).Encode(user)
	return nil

}

func getUserData(userName string) (User, error) {

	userList, err := loadUsers()
	if err != nil {
		record.Error("(load user list)", err)
	}

	user, ok := userList[userName]
	if !ok {
		return User{}, fmt.Errorf("user %v not found", userName)
	}

	return user, nil

}