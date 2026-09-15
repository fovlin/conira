package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
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
		record.Warn("(open resourse directory)", err)
	}
	defer resourseRoot.Close()

	filePath, _ := strings.CutPrefix(r.RequestURI, "/")
	file, err := resourseRoot.Open(filePath)
	if os.IsNotExist(err) {
		http.Redirect(w, r, "/status/404.html", http.StatusSeeOther)
	} else if err != nil {
		record.Warn("(open resourse file)", err)
	}
	defer file.Close()

	setMime(filePath, w)
	io.Copy(w, file)

}

func login(w http.ResponseWriter) {
	var file []byte
	var err error
	if config.Verify {
		file, err = os.ReadFile(loginFile)
		if err != nil {
			record.Error("(read login file)", err)
		}
	} else {
		file, err = os.ReadFile(loginNoVerifyFile)
		if err != nil {
			record.Error("(read login file)", err)
		}
	}

	w.Write(file)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	httpUserName := r.FormValue("user-name")

	if !config.Verify {
		setUserData(httpUserName, w)
	}

	httpPassword := r.FormValue("password")
	if httpUserName == "" || httpPassword == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user, err := loadUserData(httpUserName)
	if err != nil {
		record.Warn("load user data:", err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	httpPasswdHash, err := toSha256(httpPassword)
	if err != nil {
		record.Warn("to sha256:", err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if user.PasswordHash != httpPasswdHash {
		record.Warn("password verification failed:", "user:", httpUserName)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	setUserData(user.UserName, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {

	if !validCookie(r) {
		login(w)
		return
	}

	indexData, err := os.ReadFile(indexFile)
	if err != nil {
		record.Warn("read index file", err)
	}

	itemData, err := os.ReadFile(itemFile)
	if err != nil {
		record.Warn("read item file", err)
	}

	messageList, err := loadMessages()
	if err != nil {
		record.Warn("load message list: ", err)
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

func submitHandler(w http.ResponseWriter, r *http.Request) {

	if !validCookie(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
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

	message := Message{
		Name:    userName,
		Content: content,
		Time:    time.Now().Format(time.DateTime),
	}

	err = saveMessages(message, messageFile)
	if err != nil {
		record.Warn("save message:", err)
	}

	record.Info("new message:", "user:", message.Name, "address:", r.RemoteAddr)

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
		record.Warn("(parse cookie)", err)
	}
	tokenCookie, err := r.Cookie("Token")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		record.Warn("(parse cookie)", err)
	}

	nameCookie.MaxAge = -1
	tokenCookie.MaxAge = -1

	http.SetCookie(w, nameCookie)
	http.SetCookie(w, tokenCookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func validCookie(r *http.Request) bool {

	nameCookie, err := r.Cookie("UserName")
	if err != nil {
		return false
	}

	if !config.Verify {
		return true
	}

	tokenCookie, err := r.Cookie("Token")
	if err != nil {
		return false
	}

	user, err := loadUserData(nameCookie.Value)
	if err != nil {
		record.Warn("load user data:", err)
		return false
	}

	expiresTime, err := time.Parse(time.DateTime, user.Expires)
	if err != nil {
		record.Warn("parse expires time:", err)
		return false
	}

	if user.Token != tokenCookie.Value || !time.Now().Before(expiresTime) {
		return false
	}
	return true

}

func loadUserData(userName string) (User, error) {

	userList, err := loadUserList()
	if err != nil {
		return User{}, err
	}

	user, ok := userList[userName]
	if !ok {
		return User{}, fmt.Errorf("user %v not found", userName)
	}

	return user, nil

}

func setMime(fileName string, w http.ResponseWriter) {
	ext := path.Ext(fileName)
	if mimeType := mime.TypeByExtension(ext); len(mimeType) != 0 {
		w.Header().Set("Content-Type", mimeType)
	}
}

func setUserData(userName string, w http.ResponseWriter) error {

	user, err := loadUserData(userName)
	if err != nil {
		return err
	}

	token := make([]byte, 16)
	token, err = hex.DecodeString(user.Token)
	if err != nil {
		return err
	}

	user.Expires = time.Now().Add(time.Hour * 24 * 3).Format(time.DateTime)
	_, err = rand.Reader.Read(token)
	if err != nil {
		return err
	}

	user.Token = hex.EncodeToString(token)

	nameCookie := &http.Cookie{
		Name:    "UserName",
		Value:   user.UserName,
		Secure:  true,
		Expires: time.Now().Add(time.Hour * 24 * 3),
	}

	tokenCookie := &http.Cookie{
		Name:    "Token",
		Value:   user.Token,
		Secure:  true,
		Expires: time.Now().Add(time.Hour * 24 * 3),
	}

	http.SetCookie(w, tokenCookie)
	http.SetCookie(w, nameCookie)

	if config.Verify {
		if err := saveUserData(user); err != nil {
			record.Warn("save user data:", err)
		}
	}

	return nil

}
