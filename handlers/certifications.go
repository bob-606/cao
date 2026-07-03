package handlers

import (
	"log"
	"net/http"
	"time"

	"logbook/db"
	"logbook/middleware"
)

func CertificationsIndex(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		name := r.FormValue("name")
		typ := r.FormValue("type")
		category := r.FormValue("category")
		aircraftType := r.FormValue("aircraftType")
		issueDate := r.FormValue("issueDate")
		expiryDate := r.FormValue("expiryDate")
		authority := r.FormValue("authority")
		scope := r.FormValue("scope")

		if name == "" {
			http.Redirect(w, r, "/certifications", http.StatusSeeOther)
			return
		}

		id := db.GenerateID()
		_, err := db.DB.Exec(
			"INSERT INTO certifications (id, name, type, category, aircraft_type, issue_date, expiry_date, authority, scope, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			id, name, orDefault(typ, "license"), nullIfEmpty(category), nullIfEmpty(aircraftType),
			nullIfEmpty(issueDate), nullIfEmpty(expiryDate), orDefault(authority, "EASA"), nullIfEmpty(scope), userID,
		)
		if err != nil {
			log.Printf("Insert certification error: %v", err)
		}
		http.Redirect(w, r, "/certifications", http.StatusSeeOther)
		return
	}

	rows, err := db.DB.Query(
		"SELECT id, name, type, category, aircraft_type, issue_date, expiry_date, authority, scope FROM certifications WHERE user_id = ? ORDER BY expiry_date",
		userID,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type cert struct {
		ID, Name, Type string
		Category       *string
		AircraftType   *string
		IssueDate      *string
		ExpiryDate     *string
		Authority      string
		Scope          *string
		Status         string
	}
	var certifications []cert
	now := time.Now()
	for rows.Next() {
		var c cert
		rows.Scan(&c.ID, &c.Name, &c.Type, &c.Category, &c.AircraftType,
			&c.IssueDate, &c.ExpiryDate, &c.Authority, &c.Scope)
		if c.ExpiryDate != nil {
			exp, err := time.Parse("2006-01-02", *c.ExpiryDate)
			if err == nil {
				if exp.Before(now) {
					c.Status = "expired"
				} else if exp.Before(now.Add(30*24*time.Hour)) {
					c.Status = "expiring"
				} else {
					c.Status = "valid"
				}
			}
		}
		certifications = append(certifications, c)
	}

	var expired, expiringSoon []cert
	for _, c := range certifications {
		switch c.Status {
		case "expired":
			expired = append(expired, c)
		case "expiring":
			expiringSoon = append(expiringSoon, c)
		}
	}

	// Get supervision stats
	var totalMinutes, supervisedMinutes int64
	db.DB.QueryRow(`
		SELECT COALESCE(SUM(total_time),0), COALESCE(SUM(supervised_time),0)
		FROM maintenance_records WHERE user_id = ?`, userID).Scan(&totalMinutes, &supervisedMinutes)

	supervisionRatio := 0.0
	if totalMinutes > 0 {
		supervisionRatio = float64(supervisedMinutes) / float64(totalMinutes) * 100
	}

	render(w, r, "certifications/index", map[string]interface{}{
		"Certifications":    certifications,
		"Expired":           expired,
		"ExpiringSoon":      expiringSoon,
		"TotalMinutes":      totalMinutes,
		"SupervisedMinutes": supervisedMinutes,
		"SupervisionRatio":  supervisionRatio,
	})
}

func CertificationsDelete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	id := r.PathValue("id")

	db.DB.Exec("DELETE FROM certifications WHERE id = ? AND user_id = ?", id, userID)
	http.Redirect(w, r, "/certifications", http.StatusSeeOther)
}
