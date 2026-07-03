package handlers

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"logbook/db"
)

var estoniaAuthSessions sync.Map

type estoniaAuthSession struct {
	Provider    string
	SessionID   string
	Hash        string
	HashType    string
	Challenge   []byte
	PhoneNumber string
	NationalID  string
	Status      string
	Error       string
	GivenName   string
	Surname     string
	IDCode      string
}

func EstoniaMIDPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "auth/estonia-mid", nil)
}

func EstoniaSIDPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "auth/estonia-sid", nil)
}

func EstoniaMIDInit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	idCode := strings.TrimSpace(r.FormValue("id_code"))
	phone := strings.TrimSpace(r.FormValue("phone"))

	if idCode == "" || phone == "" {
		render(w, r, "auth/estonia-mid", map[string]interface{}{
			"Error": "Both ID code and phone number are required",
		})
		return
	}

	if !strings.HasPrefix(phone, "+") {
		phone = "+372" + phone
	}

	challenge, hashB64, hashType := generateEstoniaChallenge()

	host := os.Getenv("MID_HOST")
	if host == "" {
		host = "https://tsp.demo.sk.ee/mid-api"
	}

	rpUUID := os.Getenv("MID_RP_UUID")
	if rpUUID == "" {
		rpUUID = "00000000-0000-0000-0000-000000000000"
	}

	rpName := os.Getenv("MID_RP_NAME")
	if rpName == "" {
		rpName = "DEMO"
	}

	body := map[string]interface{}{
		"relyingPartyUUID":     rpUUID,
		"relyingPartyName":     rpName,
		"phoneNumber":          phone,
		"nationalIdentityNumber": idCode,
		"hash":                 hashB64,
		"hashType":             hashType,
		"language":             "EST",
	}

	bodyJSON, _ := json.Marshal(body)
	resp, err := http.Post(host+"/authentication", "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		log.Printf("MID init error: %v", err)
		render(w, r, "auth/estonia-mid", map[string]interface{}{
			"Error": "Failed to connect to Mobile-ID service",
		})
		return
	}
	defer resp.Body.Close()

	var result struct {
		SessionID string `json:"sessionID"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.SessionID == "" {
		log.Printf("MID init decode error: %v", err)
		render(w, r, "auth/estonia-mid", map[string]interface{}{
			"Error": "Invalid response from Mobile-ID service",
		})
		return
	}

	sessionToken := db.GenerateID()
	estoniaAuthSessions.Store(sessionToken, &estoniaAuthSession{
		Provider:    "mid",
		SessionID:   result.SessionID,
		Hash:        hashB64,
		HashType:    hashType,
		Challenge:   challenge,
		PhoneNumber: phone,
		NationalID:  idCode,
		Status:      "PENDING",
	})

	render(w, r, "auth/estonia-verify", map[string]interface{}{
		"Provider":     "Mobile-ID",
		"SessionToken": sessionToken,
		"PollURL":      "/auth/mid/status/" + sessionToken,
	})
}

func EstoniaSIDInit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	idCode := strings.TrimSpace(r.FormValue("id_code"))
	if idCode == "" {
		render(w, r, "auth/estonia-sid", map[string]interface{}{
			"Error": "ID code is required",
		})
		return
	}

	challenge, hashB64, hashType := generateEstoniaChallenge()

	host := os.Getenv("SID_HOST")
	if host == "" {
		host = "https://sid.demo.sk.ee/smart-id-rp/v2"
	}

	rpUUID := os.Getenv("SID_RP_UUID")
	if rpUUID == "" {
		rpUUID = "00000000-0000-4000-8000-000000000000"
	}

	rpName := os.Getenv("SID_RP_NAME")
	if rpName == "" {
		rpName = "DEMO"
	}

	body := map[string]interface{}{
		"relyingPartyUUID": rpUUID,
		"relyingPartyName": rpName,
		"certificateLevel": "QUALIFIED",
		"hash":             hashB64,
		"hashType":         hashType,
		"allowedInteractionsOrder": []map[string]interface{}{
			{"type": "displayTextAndPIN", "displayText60": "Log in"},
		},
	}

	bodyJSON, _ := json.Marshal(body)

	semanticID := "PNOEE-" + idCode
	req, _ := http.NewRequest("POST", host+"/authentication/etsi/"+semanticID, bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("SID init error: %v", err)
		render(w, r, "auth/estonia-sid", map[string]interface{}{
			"Error": "Failed to connect to Smart-ID service",
		})
		return
	}
	defer resp.Body.Close()

	var result struct {
		SessionID string `json:"sessionID"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.SessionID == "" {
		log.Printf("SID init decode error: %v", err)
		render(w, r, "auth/estonia-sid", map[string]interface{}{
			"Error": "Invalid response from Smart-ID service",
		})
		return
	}

	sessionToken := db.GenerateID()
	estoniaAuthSessions.Store(sessionToken, &estoniaAuthSession{
		Provider:   "sid",
		SessionID:  result.SessionID,
		Hash:       hashB64,
		HashType:   hashType,
		Challenge:  challenge,
		NationalID: idCode,
		Status:     "PENDING",
	})

	render(w, r, "auth/estonia-verify", map[string]interface{}{
		"Provider":     "Smart-ID",
		"SessionToken": sessionToken,
		"PollURL":      "/auth/sid/status/" + sessionToken,
	})
}

func EstoniaMIDStatus(w http.ResponseWriter, r *http.Request) {
	sessionToken := r.PathValue("token")
	v, ok := estoniaAuthSessions.Load(sessionToken)
	if !ok {
		json.NewEncoder(w).Encode(map[string]string{"status": "ERROR", "error": "Session not found"})
		return
	}

	s := v.(*estoniaAuthSession)
	if s.Status != "PENDING" {
		json.NewEncoder(w).Encode(map[string]string{"status": s.Status, "error": s.Error})
		return
	}

	host := os.Getenv("MID_HOST")
	if host == "" {
		host = "https://tsp.demo.sk.ee/mid-api"
	}

	resp, err := http.Get(host + "/authentication/session/" + s.SessionID + "?timeoutMs=3000")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
		return
	}
	defer resp.Body.Close()

	var status struct {
		State   string `json:"state"`
		Result  string `json:"result"`
		Cert    string `json:"cert"`
		Sign    struct {
			Value     string `json:"value"`
			Algorithm string `json:"algorithm"`
		} `json:"signature"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
		return
	}

	if status.State == "RUNNING" {
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
		return
	}

	if status.State == "COMPLETE" && status.Result == "OK" {
		givenName, surname, idCode, err := parseEstoniaCert(status.Cert)
		if err != nil {
			log.Printf("MID cert parse error: %v", err)
			s.Status = "ERROR"
			s.Error = "Failed to parse certificate"
			json.NewEncoder(w).Encode(map[string]string{"status": "ERROR", "error": "Certificate parse failed"})
			return
		}

		s.GivenName = givenName
		s.Surname = surname
		s.IDCode = idCode

		userID := findOrCreateEstoniaUser(givenName, surname, idCode)
		s.Status = "COMPLETE"

		json.NewEncoder(w).Encode(map[string]string{
			"status":  "COMPLETE",
			"userID":  userID,
		})
		return
	}

	s.Status = "ERROR"
	s.Error = status.Result
	json.NewEncoder(w).Encode(map[string]string{"status": "ERROR", "error": status.Result})
}

func EstoniaSIDStatus(w http.ResponseWriter, r *http.Request) {
	sessionToken := r.PathValue("token")
	v, ok := estoniaAuthSessions.Load(sessionToken)
	if !ok {
		json.NewEncoder(w).Encode(map[string]string{"status": "ERROR", "error": "Session not found"})
		return
	}

	s := v.(*estoniaAuthSession)
	if s.Status != "PENDING" {
		json.NewEncoder(w).Encode(map[string]string{"status": s.Status, "error": s.Error})
		return
	}

	host := os.Getenv("SID_HOST")
	if host == "" {
		host = "https://sid.demo.sk.ee/smart-id-rp/v2"
	}

	resp, err := http.Get(host + "/session/" + s.SessionID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
		return
	}
	defer resp.Body.Close()

	var status struct {
		State  string `json:"state"`
		Result string `json:"result"`
		Cert   struct {
			Value string `json:"value"`
		} `json:"cert"`
		Sign struct {
			Value     string `json:"value"`
			Algorithm string `json:"algorithm"`
		} `json:"signature"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
		return
	}

	if status.State == "RUNNING" {
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
		return
	}

	if status.State == "COMPLETE" && status.Result == "OK" {
		givenName, surname, idCode, err := parseEstoniaCert(status.Cert.Value)
		if err != nil {
			log.Printf("SID cert parse error: %v", err)
			s.Status = "ERROR"
			s.Error = "Failed to parse certificate"
			json.NewEncoder(w).Encode(map[string]string{"status": "ERROR", "error": "Certificate parse failed"})
			return
		}

		s.GivenName = givenName
		s.Surname = surname
		s.IDCode = idCode

		userID := findOrCreateEstoniaUser(givenName, surname, idCode)
		s.Status = "COMPLETE"

		json.NewEncoder(w).Encode(map[string]string{
			"status":  "COMPLETE",
			"userID":  userID,
		})
		return
	}

	s.Status = "ERROR"
	s.Error = status.Result
	json.NewEncoder(w).Encode(map[string]string{"status": "ERROR", "error": status.Result})
}

func EstoniaComplete(w http.ResponseWriter, r *http.Request) {
	sessionToken := r.PathValue("token")
	v, ok := estoniaAuthSessions.Load(sessionToken)
	if !ok {
		http.Redirect(w, r, "/auth/signin", http.StatusSeeOther)
		return
	}

	s := v.(*estoniaAuthSession)
	if s.Status != "COMPLETE" {
		http.Redirect(w, r, "/auth/signin", http.StatusSeeOther)
		return
	}

	userID := findOrCreateEstoniaUser(s.GivenName, s.Surname, s.IDCode)
	createSession(w, userID)
	estoniaAuthSessions.Delete(sessionToken)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func generateEstoniaChallenge() (challenge []byte, hashB64, hashType string) {
	challenge = make([]byte, 64)
	rand.Read(challenge)
	hash := sha256.Sum256(challenge)
	hashB64 = base64.StdEncoding.EncodeToString(hash[:])
	return challenge, hashB64, "SHA256"
}

func parseEstoniaCert(certB64 string) (givenName, surname, idCode string, err error) {
	certDER, err := base64.StdEncoding.DecodeString(certB64)
	if err != nil {
		return "", "", "", fmt.Errorf("base64 decode: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		block, _ := pem.Decode([]byte(certB64))
		if block != nil {
			cert, err = x509.ParseCertificate(block.Bytes)
		}
		if err != nil {
			return "", "", "", fmt.Errorf("parse cert: %w", err)
		}
	}

	for _, attr := range cert.Subject.Names {
		switch {
		case attr.Type.Equal([]int{2, 5, 4, 42}):
			givenName = fmt.Sprintf("%v", attr.Value)
		case attr.Type.Equal([]int{2, 5, 4, 4}):
			surname = fmt.Sprintf("%v", attr.Value)
		case attr.Type.Equal([]int{2, 5, 4, 5}):
			idCode = fmt.Sprintf("%v", attr.Value)
			idCode = strings.TrimPrefix(idCode, "PNOEE-")
			idCode = strings.TrimPrefix(idCode, "PNOLT-")
		}
	}

	if givenName == "" || surname == "" {
		cn := cert.Subject.CommonName
		parts := strings.Split(cn, ",")
		if len(parts) >= 3 {
			if surname == "" {
				surname = strings.TrimSpace(parts[0])
			}
			if givenName == "" {
				givenName = strings.TrimSpace(parts[1])
			}
			if idCode == "" {
				idCode = strings.TrimSpace(parts[2])
			}
		}
	}

	return givenName, surname, idCode, nil
}

func verifyEstoniaSignature(certB64, signatureB64, hashB64 string) error {
	certDER, err := base64.StdEncoding.DecodeString(certB64)
	if err != nil {
		return err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return err
	}

	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return err
	}

	hash, err := base64.StdEncoding.DecodeString(hashB64)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	hasher.Write(hash)
	digest := hasher.Sum(nil)

	return cert.CheckSignature(x509.ECDSAWithSHA256, digest, signature)
}

func findOrCreateEstoniaUser(givenName, surname, idCode string) string {
	email := "ee-" + idCode + "@logbook"
	userName := givenName + " " + surname

	var userID string
	err := db.DB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err == nil {
		return userID
	}

	userID = db.GenerateID()
	_, err = db.DB.Exec(
		"INSERT INTO users (id, name, email, password) VALUES (?, ?, ?, ?)",
		userID, userName, email, "",
	)
	if err != nil {
		log.Printf("Create Estonia user error: %v", err)
		return ""
	}
	return userID
}

func init() {
	estoniaAuthSessions = sync.Map{}
}
