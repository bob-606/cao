package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

const CSRFTokenKey contextKey = "csrf_token"

var csrfCookieName = "csrf_token"

func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			token := ""
			cookie, err := r.Cookie(csrfCookieName)
			if err == nil {
				token = cookie.Value
			} else {
				token = generateCSRFToken()
				http.SetCookie(w, &http.Cookie{
					Name:     csrfCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}
			ctx := context.WithValue(r.Context(), CSRFTokenKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		cookie, err := r.Cookie(csrfCookieName)
		if err != nil {
			http.Error(w, "CSRF cookie missing", http.StatusForbidden)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		formToken := r.FormValue("_csrf")
		if formToken == "" {
			http.Error(w, "CSRF token missing", http.StatusForbidden)
			return
		}

		if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formToken)) != 1 {
			http.Error(w, "CSRF token mismatch", http.StatusForbidden)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     csrfCookieName,
			Value:    cookie.Value,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		ctx := context.WithValue(r.Context(), CSRFTokenKey, cookie.Value)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetCSRFToken(r *http.Request) string {
	v, _ := r.Context().Value(CSRFTokenKey).(string)
	return v
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
