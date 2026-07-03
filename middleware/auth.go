package middleware

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"logbook/db"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const UserNameKey contextKey = "user_name"
const UserEmailKey contextKey = "user_email"

var sessionSecret string

func init() {
	sessionSecret = os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "dev-secret-change-me"
	}
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/auth/signin", http.StatusSeeOther)
			return
		}

		var userID, userName, userEmail string
		err = db.DB.QueryRow(
			`SELECT u.id, u.name, u.email FROM sessions s
			 JOIN users u ON u.id = s.user_id
			 WHERE s.id = ? AND s.created_at > ?`,
			cookie.Value, time.Now().Add(-24*7*time.Hour),
		).Scan(&userID, &userName, &userEmail)

		if err == sql.ErrNoRows {
			http.SetCookie(w, &http.Cookie{
				Name: "session_id", Value: "", MaxAge: -1, Path: "/", SameSite: http.SameSiteLaxMode,
			})
			http.Redirect(w, r, "/auth/signin", http.StatusSeeOther)
			return
		}
		if err != nil {
			log.Printf("Session query error: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, UserNameKey, userName)
		ctx = context.WithValue(ctx, UserEmailKey, userEmail)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(r *http.Request) string {
	v, _ := r.Context().Value(UserIDKey).(string)
	return v
}

func GetUserName(r *http.Request) string {
	v, _ := r.Context().Value(UserNameKey).(string)
	return v
}

func GetUserEmail(r *http.Request) string {
	v, _ := r.Context().Value(UserEmailKey).(string)
	return v
}
