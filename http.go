package main

import (
	"bytes"
	"net/http"
	"os"
	"time"

	"acovia.net/record"
)

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

	userName := r.FormValue("user-name")
	password := r.FormValue("password")
	if userName == "" || password == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	userList, err := loadUser()
	if err != nil {
		record.Warn("(load user list)", err)
		return
	}

	user, ok := userList[userName]

	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if user.Password != password {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	cookie := &http.Cookie{
		Name:    "IsLogin",
		Value:   user.UserName,
		Secure:  true,
		Expires: time.Now().Add(time.Hour * 24 * 3),
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func mainHandler(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("IsLogin")
	if err == http.ErrNoCookie {
		login(w)
		return
	} else if err != nil {
		record.Warn("(check cookie)", err)
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

	if len(messageList) > maxHttpJsonLength {
		messageList = messageList[:maxHttpJsonLength]
	}

	allMessage := []byte{}

	for _, Message := range messageList {
		aMessageData := itemData
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ name }}"), []byte(Message.Name))
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ time }}"), []byte(Message.Time))
		aMessageData = bytes.ReplaceAll(aMessageData, []byte("{{ content }}"), []byte(Message.Content))
		allMessage = append(allMessage, aMessageData...)
	}

	if len(allMessage) == 0 {
		allMessage = []byte("<p style=\"text-align: center\">None message</p>")
	}

	userName := []byte(cookie.Value)
	indexData = bytes.ReplaceAll(indexData, []byte("{{ user-name }}"), userName)
	indexData = bytes.ReplaceAll(indexData, []byte("{{ messages }}"), allMessage)
	w.Write(indexData)

}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	cookie, err := r.Cookie("IsLogin")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	name := cookie.Value

	content := r.FormValue("content")
	if content == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	msgs, _ := loadMessages()
	msg := Message{
		Name:    name,
		Content: content,
		Time:    time.Now().Format(time.DateTime),
	}
	msgs = append([]Message{msg}, msgs...)

	if len(msgs) > maxJsonLength {
		msgs = msgs[:maxJsonLength]
	}

	saveMessages(msgs, messageFile)

	record.Info("(new message)", "user:", "\"" + msg.Name + "\"", "address:", "\"" + r.RemoteAddr + "\"")

	http.Redirect(w, r, "/", http.StatusSeeOther)
	
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	cookie, err := r.Cookie("IsLogin")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		record.Error("(parse cookie)", err)
	}

	cookie.MaxAge = -1

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}