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

	user, err := loadUserData(httpUserName)
	if err != nil {
		record.Warn("(load user data)", err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
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

	if !time.Now().Before(expiresTime) || len(token) != 16 {

		user.Expires = time.Now().Add(time.Hour * 24 * 3).Format(time.DateTime)
		_, err := rand.Reader.Read(token)
		if err != nil {
			record.Error("create user token:", err)
		}

	}

	httpPasswdHash, err := toSha256(httpPassword)

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

	user.Token = hex.EncodeToString(token)

	if err = saveUserData(user); err != nil {
		record.Error("save user data:", err)
	}

}

func indexHandler(w http.ResponseWriter, r *http.Request) {

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

	message := Message{
		Name:    userName,
		Content: content,
		Time:    time.Now().Format(time.DateTime),
	}

	err = saveMessages(message, messageFile)
	if err != nil {
		record.Error("save message:", err)
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

	user, err := loadUserData(nameCookie.Value)
	if err != nil {
		record.Error("load user data:", err)
		return false
	}

	if user.Token != tokenCookie.Value {
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
