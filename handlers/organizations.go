package handlers

import (
	"log"
	"net/http"

	"logbook/db"
	"logbook/middleware"
)

func OrganizationsIndex(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		name := r.FormValue("name")
		typ := r.FormValue("type")
		easaPart := r.FormValue("easaPartNumber")
		address := r.FormValue("address")
		country := r.FormValue("country")

		if name == "" {
			http.Redirect(w, r, "/organizations", http.StatusSeeOther)
			return
		}

		id := db.GenerateID()
		_, err := db.DB.Exec(
			"INSERT INTO organizations (id, name, type, easa_part_number, address, country, owner_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
			id, name, orDefault(typ, "camo"), nullIfEmpty(easaPart), nullIfEmpty(address), orDefault(country, "EE"), userID,
		)
		if err != nil {
			log.Printf("Insert organization error: %v", err)
		}
		http.Redirect(w, r, "/organizations", http.StatusSeeOther)
		return
	}

	rows, err := db.DB.Query(`
		SELECT o.id, o.name, o.type, o.easa_part_number, o.address, o.country,
			(SELECT COUNT(*) FROM maintenance_records m WHERE m.organization_id = o.id) as record_count
		FROM organizations o WHERE o.owner_id = ? ORDER BY o.name
	`, userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type org struct {
		ID, Name, Type string
		EASAPartNumber *string
		Address        *string
		Country        string
		RecordCount    int
	}
	var organizations []org
	for rows.Next() {
		var o org
		rows.Scan(&o.ID, &o.Name, &o.Type, &o.EASAPartNumber, &o.Address, &o.Country, &o.RecordCount)
		organizations = append(organizations, o)
	}

	render(w, r, "organizations/index", map[string]interface{}{
		"Organizations": organizations,
	})
}

func OrganizationsDelete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	id := r.PathValue("id")

	db.DB.Exec("DELETE FROM organizations WHERE id = ? AND owner_id = ?", id, userID)
	http.Redirect(w, r, "/organizations", http.StatusSeeOther)
}
