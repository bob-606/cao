package handlers

import (
	"log"
	"net/http"
	"strings"

	"logbook/db"
	"logbook/middleware"
)

func AircraftIndex(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		reg := stringsToUpper(r.FormValue("registration"))
		typ := r.FormValue("type")
		variant := r.FormValue("variant")
		model := r.FormValue("model")
		serial := r.FormValue("serialNumber")
		year := parseInt(r.FormValue("yearOfManufacture"))
		category := r.FormValue("category")
		engineType := r.FormValue("engineType")
		apuType := r.FormValue("apuType")
		propType := r.FormValue("propType")
		mtow := parseInt(r.FormValue("mtow"))
		maxPax := parseInt(r.FormValue("maxPax"))

		if reg == "" || typ == "" {
			http.Redirect(w, r, "/aircraft", http.StatusSeeOther)
			return
		}

		if category == "" {
			category = "airplane"
		}

		id := db.GenerateID()
		var y, m, mp *int64
		if year > 0 {
			y = &year
		}
		if mtow > 0 {
			m = &mtow
		}
		if maxPax > 0 {
			mp = &maxPax
		}
		_, err := db.DB.Exec(
			`INSERT INTO aircraft (id, registration, type, variant, model, serial_number, year_of_manufacture,
			 category, engine_type, apu_type, prop_type, mtow, max_pax, owner_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, reg, typ, nullIfEmpty(variant), nullIfEmpty(model), nullIfEmpty(serial), y,
			category, nullIfEmpty(engineType), nullIfEmpty(apuType), nullIfEmpty(propType), m, mp, userID,
		)
		if err != nil {
			log.Printf("Insert aircraft error: %v", err)
		}
		http.Redirect(w, r, "/aircraft", http.StatusSeeOther)
		return
	}

	rows, err := db.DB.Query(`
		SELECT a.id, a.registration, a.type, a.variant, a.model, a.serial_number,
			a.year_of_manufacture, a.category, a.engine_type, a.apu_type, a.prop_type,
			a.mtow, a.max_pax,
			(SELECT COUNT(*) FROM maintenance_records m WHERE m.aircraft_id = a.id) as record_count
		FROM aircraft a WHERE a.owner_id = ? ORDER BY a.registration
	`, userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ac struct {
		ID, Registration, Type, Category string
		Variant, Model, SerialNumber     *string
		YearOfManufacture, MTOW, MaxPax  *int64
		EngineType, APUType, PropType    *string
		RecordCount                      int
	}
	var aircraft []ac
	for rows.Next() {
		var a ac
		rows.Scan(&a.ID, &a.Registration, &a.Type, &a.Variant, &a.Model, &a.SerialNumber,
			&a.YearOfManufacture, &a.Category, &a.EngineType, &a.APUType, &a.PropType,
			&a.MTOW, &a.MaxPax, &a.RecordCount)
		aircraft = append(aircraft, a)
	}

	render(w, r, "aircraft/index", map[string]interface{}{
		"Aircraft": aircraft,
	})
}

func AircraftDelete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	id := r.PathValue("id")

	db.DB.Exec("DELETE FROM aircraft WHERE id = ? AND owner_id = ?", id, userID)
	http.Redirect(w, r, "/aircraft", http.StatusSeeOther)
}

func stringsToUpper(s string) string {
	return strings.ToUpper(s)
}
