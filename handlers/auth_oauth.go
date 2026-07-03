package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"logbook/db"

	"golang.org/x/oauth2"
)

var googleOAuthConfig *oauth2.Config
var githubOAuthConfig *oauth2.Config

func init() {
	googleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("BASE_URL") + "/auth/google/callback",
		Scopes:       []string{"email", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	githubOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("BASE_URL") + "/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}
}

func OAuthLogin(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	var config *oauth2.Config
	switch provider {
	case "google":
		config = googleOAuthConfig
	case "github":
		config = githubOAuthConfig
	default:
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	state := generateOAuthState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})

	url := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func OAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	var config *oauth2.Config
	switch provider {
	case "google":
		config = googleOAuthConfig
	case "github":
		config = githubOAuthConfig
	default:
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "State cookie missing", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name: "oauth_state", Value: "", MaxAge: -1, Path: "/",
		SameSite: http.SameSiteLaxMode,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code missing", http.StatusBadRequest)
		return
	}

	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("OAuth token exchange error: %v", err)
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	email, name, providerUserID, err := getOAuthUserInfo(provider, token.AccessToken)
	if err != nil {
		log.Printf("OAuth user info error: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	userID := findOrCreateOAuthUser(provider, providerUserID, email, name)
	if userID == "" {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	createSession(w, userID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func generateOAuthState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func getOAuthUserInfo(provider, accessToken string) (email, name, id string, err error) {
	switch provider {
	case "google":
		return getGoogleUserInfo(accessToken)
	case "github":
		return getGitHubUserInfo(accessToken)
	default:
		return "", "", "", fmt.Errorf("unknown provider: %s", provider)
	}
}

func getGoogleUserInfo(accessToken string) (string, string, string, error) {
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	var info struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", "", "", err
	}
	return info.Email, info.Name, info.ID, nil
}

func getGitHubUserInfo(accessToken string) (string, string, string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	var info struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", "", "", err
	}

	if info.Email == "" {
		email, err := getGitHubPrimaryEmail(accessToken)
		if err != nil {
			return "", "", "", err
		}
		info.Email = email
	}

	if info.Name == "" {
		info.Name = info.Login
	}

	return info.Email, info.Name, fmt.Sprintf("%d", info.ID), nil
}

func getGitHubPrimaryEmail(accessToken string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}
	return "", fmt.Errorf("no email found")
}

func findOrCreateOAuthUser(provider, providerUserID, email, name string) string {
	var userID string
	err := db.DB.QueryRow(
		"SELECT user_id FROM oauth_accounts WHERE provider = ? AND provider_user_id = ?",
		provider, providerUserID,
	).Scan(&userID)
	if err == nil {
		return userID
	}

	err = db.DB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err == nil {
		db.DB.Exec(
			"INSERT INTO oauth_accounts (id, user_id, provider, provider_user_id) VALUES (?, ?, ?, ?)",
			db.GenerateID(), userID, provider, providerUserID,
		)
		return userID
	}

	userID = db.GenerateID()
	_, err = db.DB.Exec(
		"INSERT INTO users (id, name, email, password) VALUES (?, ?, ?, ?)",
		userID, name, email, "",
	)
	if err != nil {
		log.Printf("Create OAuth user error: %v", err)
		return ""
	}

	db.DB.Exec(
		"INSERT INTO oauth_accounts (id, user_id, provider, provider_user_id) VALUES (?, ?, ?, ?)",
		db.GenerateID(), userID, provider, providerUserID,
	)
	return userID
}
