package api

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"logbook/assets"
	"logbook/db"
	"logbook/handlers"
	"logbook/middleware"
)

var (
	once sync.Once
	mux  http.Handler
)

func initMux() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		if os.Getenv("VERCEL") != "" {
			dbPath = "/tmp/cao.db"
		} else {
			dbPath = "./data/cao.db"
		}
	}
	db.Init(dbPath)

	m := http.NewServeMux()

	staticFS, _ := fs.Sub(assets.FS, "static")
	m.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	m.Handle("GET /auth/signin", middleware.CSRF(http.HandlerFunc(handlers.SignInPage)))
	m.Handle("POST /auth/signin", middleware.CSRF(http.HandlerFunc(handlers.SignIn)))
	m.Handle("GET /auth/register", middleware.CSRF(http.HandlerFunc(handlers.RegisterPage)))
	m.Handle("POST /auth/register", middleware.CSRF(http.HandlerFunc(handlers.Register)))
	m.HandleFunc("GET /auth/demo", handlers.DemoLogin)
	m.HandleFunc("GET /auth/signout", handlers.SignOut)

	m.Handle("GET /auth/mid", middleware.CSRF(http.HandlerFunc(handlers.EstoniaMIDPage)))
	m.Handle("POST /auth/mid/init", middleware.CSRF(http.HandlerFunc(handlers.EstoniaMIDInit)))
	m.HandleFunc("GET /auth/mid/status/{token}", handlers.EstoniaMIDStatus)
	m.Handle("GET /auth/sid", middleware.CSRF(http.HandlerFunc(handlers.EstoniaSIDPage)))
	m.Handle("POST /auth/sid/init", middleware.CSRF(http.HandlerFunc(handlers.EstoniaSIDInit)))
	m.HandleFunc("GET /auth/sid/status/{token}", handlers.EstoniaSIDStatus)
	m.HandleFunc("GET /auth/estonia/complete/{token}", handlers.EstoniaComplete)

	m.HandleFunc("GET /auth/{provider}/login", handlers.OAuthLogin)
	m.HandleFunc("GET /auth/{provider}/callback", handlers.OAuthCallback)

	m.Handle("GET /auth/magic", middleware.CSRF(http.HandlerFunc(handlers.MagicPage)))
	m.Handle("POST /auth/magic", middleware.CSRF(http.HandlerFunc(handlers.MagicSend)))
	m.HandleFunc("GET /auth/magic/verify", handlers.MagicVerify)

	m.Handle("GET /", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.Dashboard))))
	m.Handle("GET /maintenance", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.MaintenanceIndex))))
	m.Handle("GET /maintenance/new", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.MaintenanceNew))))
	m.Handle("POST /maintenance/new", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.MaintenanceCreate))))
	m.Handle("GET /maintenance/{id}", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.MaintenanceShow))))
	m.Handle("POST /maintenance/{id}/delete", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.MaintenanceDelete))))
	m.Handle("GET /aircraft", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.AircraftIndex))))
	m.Handle("POST /aircraft", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.AircraftIndex))))
	m.Handle("POST /aircraft/{id}/delete", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.AircraftDelete))))
	m.Handle("GET /organizations", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.OrganizationsIndex))))
	m.Handle("POST /organizations", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.OrganizationsIndex))))
	m.Handle("POST /organizations/{id}/delete", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.OrganizationsDelete))))
	m.Handle("GET /certifications", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.CertificationsIndex))))
	m.Handle("POST /certifications", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.CertificationsIndex))))
	m.Handle("POST /certifications/{id}/delete", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.CertificationsDelete))))
	m.Handle("GET /reports", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.ReportsIndex))))
	m.Handle("GET /export/pdf", middleware.Auth(middleware.CSRF(http.HandlerFunc(handlers.ExportPDF))))

	mux = m
}

func Handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handler called: %s %s", r.Method, r.URL.String())

	for _, prefix := range []string{"/api/index.go", "/api/index"} {
		if r.URL.Path == prefix {
			r.URL.Path = "/"
		} else if strings.HasPrefix(r.URL.Path, prefix+"/") {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		}
	}

	once.Do(func() {
		log.Println("Initializing CAO Logbook on Vercel")
		initMux()
	})
	mux.ServeHTTP(w, r)
}
