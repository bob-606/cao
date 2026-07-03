package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"logbook/assets"
	"logbook/middleware"
)

var templates map[string]*template.Template

func init() {
	templates = make(map[string]*template.Template)

	pages := []string{
		"auth/signin", "auth/register", "auth/magic", "auth/magic-sent",
		"auth/estonia-mid", "auth/estonia-sid", "auth/estonia-verify",
		"dashboard",
		"maintenance/index", "maintenance/new", "maintenance/show",
		"aircraft/index",
		"organizations/index",
		"certifications/index",
		"reports/index",
	}

	funcMap := template.FuncMap{
		"formatDate": func(date string) string {
			if len(date) >= 10 {
				return date[:10]
			}
			return date
		},
		"formatTime": func(m int64) string {
			h := m / 60
			min := m % 60
			if min < 0 {
				min = 0
			}
			return itoa(h) + "h " + itoa(min) + "m"
		},
		"hasPrefix":  strings.HasPrefix,
		"upper":      strings.ToUpper,
		"default": func(s, def string) string {
			if s == "" {
				return def
			}
			return s
		},
		"seq": func(n int) []int {
			r := make([]int, n)
			for i := 0; i < n; i++ {
				r[i] = i
			}
			return r
		},
		"truncate": truncate,
	}

	for _, page := range pages {
		tmpl, err := template.New("base.html").Funcs(funcMap).ParseFS(assets.FS,
			"templates/base.html",
			"templates/"+page+".html",
		)
		if err != nil {
			panic(err)
		}
		templates[page] = tmpl
	}
}

func render(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	tmpl, ok := templates[name]
	if !ok {
		http.Error(w, "Template not found: "+name, http.StatusInternalServerError)
		return
	}

	if data == nil {
		data = make(map[string]interface{})
	}
	data["CurrentPath"] = r.URL.Path

	theme := "light"
	if c, err := r.Cookie("logbook-theme"); err == nil {
		if c.Value == "nord" || c.Value == "light" || c.Value == "system" {
			theme = c.Value
		}
	}
	data["Theme"] = theme
	data["CSRFToken"] = middleware.GetCSRFToken(r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func parseInt(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		} else {
			return 0
		}
	}
	return n
}

func fmtMin(m int64) string {
	if m == 0 {
		return "0h 0m"
	}
	h := m / 60
	n := m % 60
	return itoa(h) + "h " + itoa(n) + "m"
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
