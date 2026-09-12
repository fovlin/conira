package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"

	"acovia.net/record"
)

func adminLoginHandler() (w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	httpUserName := r.FormValue("user-name")
	httpPassword := r.FormValue("password")
	if httpUserName == "" || httpPassword == "" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	user, err := loadAdminUserData(httpUserName)
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

	return
}

func adminHandler(w http.ResponseWriter, r *http.Request) {

	if !checkAdminCookie(w, r) {
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

func loadAdminUserData(userName string) (User, error) {

	userList, err := loadUserList()
	if err != nil {
		record.Error("(load user list)", err)
	}

	user, ok := userList[userName]
	if !ok {
		return User{}, fmt.Errorf("user %v not found", userName)
	}

	return user, nil

}

func checkAdminCookie(w http.ResponseWriter, r *http.Request) bool {

	nameCookie, err := r.Cookie("AdminUserName")
	if err != nil {
		record.Warn("(check admin cookie)", err)
		return false
	}

	tokenCookie, err := r.Cookie("AdminToken")
	if err != nil {
		record.Warn("(check admin cookie)", err)
		return false
	}

	user, err := loadUserData(nameCookie.Value)
	if err != nil {
		record.Error("load user data", err)
		return false
	}

	if user.Token != tokenCookie.Value {
		return false
	}

	return true

}