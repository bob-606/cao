package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"logbook/db"
)

func MagicPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "auth/magic", nil)
}

func MagicSend(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		render(w, r, "auth/magic", map[string]interface{}{
			"Error": "Email is required",
		})
		return
	}

	token := generateMagicToken()
	id := db.GenerateID()

	_, err := db.DB.Exec(
		"INSERT INTO magic_links (id, email, token, expires_at) VALUES (?, ?, ?, ?)",
		id, email, token, time.Now().Add(15*time.Minute),
	)
	if err != nil {
		log.Printf("Magic link insert error: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if err := sendMagicLink(email, token); err != nil {
		log.Printf("Send magic link error: %v", err)
		http.Error(w, "Failed to send email. Check SMTP configuration.", http.StatusInternalServerError)
		return
	}

	render(w, r, "auth/magic-sent", nil)
}

func MagicVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token missing", http.StatusBadRequest)
		return
	}

	var id, email string
	err := db.DB.QueryRow(
		"SELECT id, email FROM magic_links WHERE token = ? AND used = 0 AND expires_at > ?",
		token, time.Now(),
	).Scan(&id, &email)

	if err != nil {
		render(w, r, "auth/signin", map[string]interface{}{
			"Error": "Invalid or expired link",
		})
		return
	}

	db.DB.Exec("UPDATE magic_links SET used = 1 WHERE id = ?", id)

	var userID string
	err = db.DB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err != nil {
		userID = db.GenerateID()
		_, err = db.DB.Exec(
			"INSERT INTO users (id, name, email, password) VALUES (?, ?, ?, ?)",
			userID, email, email, "",
		)
		if err != nil {
			log.Printf("Create magic link user error: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	}

	createSession(w, userID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func generateMagicToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func sendMagicLink(email, token string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	baseURL := os.Getenv("BASE_URL")

	if host == "" || port == "" {
		return fmt.Errorf("SMTP not configured")
	}

	if from == "" {
		from = user
	}

	link := baseURL + "/auth/magic/verify?token=" + token
	subject := "Sign in to Logbook"
	body := fmt.Sprintf("Click the link to sign in:\n\n%s\n\nThis link expires in 15 minutes.", link)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", from, email, subject, body)

	addr := host + ":" + port
	auth := smtp.PlainAuth("", user, pass, host)
	return smtp.SendMail(addr, auth, from, []string{email}, []byte(msg))
}
