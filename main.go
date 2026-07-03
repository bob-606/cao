package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"

	"logbook/assets"
	"logbook/db"
	"logbook/handlers"
	"logbook/middleware"
)

func main() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/cao.db"
	}
	db.Init(dbPath)

	mux := http.NewServeMux()

	// Static files
	staticFS, _ := fs.Sub(assets.FS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Auth routes
	mux.Handle("GET /auth/signin", middleware.CSRF(http.HandlerFunc(handlers.SignInPage)))
	mux.Handle("POST /auth/signin", middleware.CSRF(http.HandlerFunc(handlers.SignIn)))
	mux.Handle("GET /auth/register", middleware.CSRF(http.HandlerFunc(handlers.RegisterPage)))
	mux.Handle("POST /auth/register", middleware.CSRF(http.HandlerFunc(handlers.Register)))
	mux.HandleFunc("GET /auth/demo", handlers.DemoLogin)
	mux.HandleFunc("GET /auth/signout", handlers.SignOut)

	// Estonia e-ID routes
	mux.Handle("GET /auth/mid", middleware.CSRF(http.HandlerFunc(handlers.EstoniaMIDPage)))
	mux.Handle("POST /auth/mid/init", middleware.CSRF(http.HandlerFunc(handlers.EstoniaMIDInit)))
	mux.HandleFunc("GET /auth/mid/status/{token}", handlers.EstoniaMIDStatus)
	mux.Handle("GET /auth/sid", middleware.CSRF(http.HandlerFunc(handlers.EstoniaSIDPage)))
	mux.Handle("POST /auth/sid/init", middleware.CSRF(http.HandlerFunc(handlers.EstoniaSIDInit)))
	mux.HandleFunc("GET /auth/sid/status/{token}", handlers.EstoniaSIDStatus)
	mux.HandleFunc("GET /auth/estonia/complete/{token}", handlers.EstoniaComplete)

	// OAuth routes
	mux.HandleFunc("GET /auth/{provider}/login", handlers.OAuthLogin)
	mux.HandleFunc("GET /auth/{provider}/callback", handlers.OAuthCallback)

	// Magic link routes
	mux.Handle("GET /auth/magic", middleware.CSRF(http.HandlerFunc(handlers.MagicPage)))
	mux.Handle("POST /auth/magic", middleware.CSRF(http.HandlerFunc(handlers.MagicSend)))
	mux.HandleFunc("GET /auth/magic/verify", handlers.MagicVerify)

	// Protected routes
	mux.Handle("GET /", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.Dashboard))))
	mux.Handle("GET /maintenance", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.MaintenanceIndex))))
	mux.Handle("GET /maintenance/new", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.MaintenanceNew))))
	mux.Handle("POST /maintenance/new", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.MaintenanceCreate))))
	mux.Handle("GET /maintenance/{id}", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.MaintenanceShow))))
	mux.Handle("POST /maintenance/{id}/delete", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.MaintenanceDelete))))
	mux.Handle("GET /aircraft", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.AircraftIndex))))
	mux.Handle("POST /aircraft", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.AircraftIndex))))
	mux.Handle("POST /aircraft/{id}/delete", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.AircraftDelete))))
	mux.Handle("GET /organizations", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.OrganizationsIndex))))
	mux.Handle("POST /organizations", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.OrganizationsIndex))))
	mux.Handle("POST /organizations/{id}/delete", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.OrganizationsDelete))))
	mux.Handle("GET /certifications", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.CertificationsIndex))))
	mux.Handle("POST /certifications", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.CertificationsIndex))))
	mux.Handle("POST /certifications/{id}/delete", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.CertificationsDelete))))
	mux.Handle("GET /reports", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.ReportsIndex))))
	mux.Handle("GET /export/pdf", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.ExportPDF))))

	addr := ":3000"
	log.Printf("CAO Logbook starting on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
