package handlers

import (
	"log"
	"net/http"
	"os"
	"time"

	"logbook/db"

	"golang.org/x/crypto/bcrypt"
)

func RegisterPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "auth/register", nil)
}

func Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if name == "" || email == "" || password == "" {
		render(w, r, "auth/register", map[string]interface{}{
			"Error": "All fields are required",
		})
		return
	}

	var exists int
	db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&exists)
	if exists > 0 {
		render(w, r, "auth/register", map[string]interface{}{
			"Error": "Email already in use",
		})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Printf("bcrypt error: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	id := db.GenerateID()
	_, err = db.DB.Exec(
		"INSERT INTO users (id, name, email, password) VALUES (?, ?, ?, ?)",
		id, name, email, string(hashed),
	)
	if err != nil {
		log.Printf("Insert user error: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	createSession(w, id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func SignInPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, "auth/signin", nil)
}

func SignIn(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	var id, hashed string
	err := db.DB.QueryRow(
		"SELECT id, password FROM users WHERE email = ?", email,
	).Scan(&id, &hashed)

	if err != nil {
		render(w, r, "auth/signin", map[string]interface{}{
			"Error": "Invalid email or password",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)); err != nil {
		render(w, r, "auth/signin", map[string]interface{}{
			"Error": "Invalid email or password",
		})
		return
	}

	createSession(w, id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func DemoLogin(w http.ResponseWriter, r *http.Request) {
	var id string
	err := db.DB.QueryRow("SELECT id FROM users WHERE email = ?", "demo@logbook").Scan(&id)
	if err != nil {
		id = db.GenerateID()
		_, err = db.DB.Exec(
			"INSERT INTO users (id, name, email, password) VALUES (?, ?, ?, ?)",
			id, "Demo Pilot", "demo@logbook", "",
		)
		if err != nil {
			log.Printf("Create demo user error: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		seedDemoData(id)
	}
	createSession(w, id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func seedDemoData(userID string) {
	ac1 := db.GenerateID()
	ac2 := db.GenerateID()
	ac3 := db.GenerateID()
	ac4 := db.GenerateID()
	ac5 := db.GenerateID()
	ac6 := db.GenerateID()
	y1, y2, y3, y4, y5 := int64(2005), int64(2010), int64(1998), int64(2015), int64(1987)
	mt1, mt2, mt3, mt4 := int64(75000), int64(9000), int64(1200), int64(525)
	mp1 := int64(180)
	ws1 := 15.0
	ew1 := int64(230)
	db.DB.Exec("INSERT INTO aircraft (id, registration, type, variant, model, serial_number, year_of_manufacture, category, engine_type, mtow, max_pax, owner_id) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)", ac1, "ES-ANS", "Airbus A320", "214", "CFM56-5B", "1234", &y1, "airplane", "turbofan", &mt1, &mp1, userID)
	db.DB.Exec("INSERT INTO aircraft (id, registration, type, variant, model, serial_number, year_of_manufacture, category, engine_type, mtow, owner_id) VALUES (?,?,?,?,?,?,?,?,?,?)", ac2, "ES-TLV", "Cessna 172R", "R", "IO-360", "172R-1234", &y2, "airplane", "piston", &mt2, userID)
	db.DB.Exec("INSERT INTO aircraft (id, registration, type, serial_number, year_of_manufacture, category, engine_type, mtow, owner_id) VALUES (?,?,?,?,?,?,?,?,?)", ac3, "ES-CMK", "Beechjet 400A", "RK-209", &y3, "airplane", "turbofan", &mt2, userID)
	db.DB.Exec("INSERT INTO aircraft (id, registration, type, model, serial_number, year_of_manufacture, category, engine_type, mtow, owner_id) VALUES (?,?,?,?,?,?,?,?,?,?)", ac4, "YR-5678", "IAR-46", "Rotax 912", "IAR-123", &y4, "airplane", "piston", &mt3, userID)
	db.DB.Exec("INSERT INTO aircraft (id, registration, type, model, category, engine_type, owner_id) VALUES (?,?,?,?,?,?,?)", ac5, "ES-TRT", "Robinson R44", "Raven II", "helicopter", "piston", userID)
	db.DB.Exec("INSERT INTO aircraft (id, registration, type, variant, serial_number, year_of_manufacture, category, mtow, wing_span_m, empty_weight_kg, owner_id) VALUES (?,?,?,?,?,?,?,?,?,?,?)", ac6, "ES-3302", "LS-4", "b", "4505", &y5, "sailplane", &mt4, &ws1, &ew1, userID)

	org1 := db.GenerateID()
	org2 := db.GenerateID()
	org3 := db.GenerateID()
	db.DB.Exec("INSERT INTO organizations (id, name, type, easa_part_number, country, owner_id) VALUES (?, ?, ?, ?, ?, ?)", org1, "Nordic CAMO OÜ", "camo", "EASA.145.0123", "EE", userID)
	db.DB.Exec("INSERT INTO organizations (id, name, type, easa_part_number, country, owner_id) VALUES (?, ?, ?, ?, ?, ?)", org2, "Tallinn Line Maintenance", "145", "EASA.145.0456", "EE", userID)
	db.DB.Exec("INSERT INTO organizations (id, name, type, country, owner_id) VALUES (?, ?, ?, ?, ?)", org3, "Estonian Air Academy", "ato", "EE", userID)

	mr1 := db.GenerateID()
	mr2 := db.GenerateID()
	mr3 := db.GenerateID()
	mr4 := db.GenerateID()
	mr5 := db.GenerateID()
	tsn1, csn1 := int64(12500), int64(8500)
	tsn2, csn2 := int64(3200), int64(4100)
	db.DB.Exec(`INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, task_card, description, work_type, maintenance_category, total_time, supervised_time, supervisor_name, supervisor_cert, organization_id, location, aircraft_tsn, aircraft_csn, remarks, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mr1, "2026-06-15", ac1, "32", "WO-2026-001", "32-41-01-01",
		"Landing gear system inspection and retraction test", "scheduled", "airframe",
		180, 60, "Toomas T.", "B1.1-00123", org1, "hangar", &tsn1, &csn1, "A320 MLG 4-year check. All ok.", userID)
	db.DB.Exec(`INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, description, work_type, maintenance_category, total_time, organization_id, location, remarks, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mr2, "2026-06-20", ac2, "71", "WO-2026-002",
		"100h inspection and oil change", "scheduled", "engine",
		120, org2, "line", "C172 100h. Oil filter changed, comp OK.", userID)
	db.DB.Exec(`INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, task_card, defect_ref, description, work_type, maintenance_category, total_time, supervised_time, supervisor_name, supervisor_cert, organization_id, location, remarks, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mr3, "2026-07-01", ac3, "24", "WO-2026-003", "24-21-01", "D-2026-001",
		"Battery replacement — Ni-Cd vented cell", "unscheduled", "electrical",
		90, 90, "Mati M.", "B2-00567", org1, "hangar", "Battery found with low capacity. Replaced per CMM.", userID)
	db.DB.Exec(`INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, description, work_type, maintenance_category, total_time, organization_id, location, component_part_number, component_serial_number, component_name, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mr4, "2026-07-05", ac4, "61", "WO-2026-004",
		"Propeller static balance", "scheduled", "airframe",
		45, org2, "line", "MTV-12-B/180-5", "P-2024-789", "Propeller assembly", userID)
	db.DB.Exec(`INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, task_card, description, work_type, maintenance_category, total_time, supervised_time, supervisor_name, supervisor_cert, organization_id, location, aircraft_tsn, aircraft_csn, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mr5, "2026-07-10", ac1, "27", "WO-2026-005", "27-11-00-01",
		"Aileron cable tension adjustment", "rectification", "airframe",
		60, 30, "Toomas T.", "B1.1-00123", org1, "hangar", &tsn2, &csn2, userID)

	mr6 := db.GenerateID()
	mr7 := db.GenerateID()
	mr8 := db.GenerateID()
	mr9 := db.GenerateID()
	mr10 := db.GenerateID()
	db.DB.Exec(`INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, description, work_type, maintenance_category, total_time, supervised_time, supervisor_name, organization_id, location, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mr6, "2026-07-12", ac5, "63", "WO-2026-006",
		"Main rotor gearbox oil analysis sample", "scheduled", "airframe",
		30, 15, "Kalle K.", org1, "hangar", userID)
	db.DB.Exec(`INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, description, work_type, maintenance_category, total_time, organization_id, location, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mr7, "2026-07-14", ac1, "21", "WO-2026-007",
		"Monthly cabin air filter check", "scheduled", "airframe",
		15, org1, "line", userID)
	db.DB.Exec(`INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, description, work_type, maintenance_category, total_time, supervised_time, supervisor_name, supervisor_cert, organization_id, location, aircraft_tsn, aircraft_csn, component_part_number, component_serial_number, component_name, remarks, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mr8, "2026-06-01", ac6, "57", "GL-2026-001",
		"Annual inspection — wing attachment and control cables", "inspection", "airframe",
		240, 120, "Peeter P.", "B1.1-00345", org1, "hangar", int64(850), int64(320),
		"LS4-WA-001", "W-2024-015", "Main wing attachment fitting", "Found one cracked bolt at rear spar. Replaced per TMM.", userID)
	db.DB.Exec(`INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, description, work_type, maintenance_category, total_time, organization_id, location, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mr9, "2026-06-15", ac6, "32",
		"Landing gear wheel bearing repack and tire check", "scheduled", "airframe",
		60, org1, "hangar", userID)
	db.DB.Exec(`INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, description, work_type, maintenance_category, total_time, organization_id, location, component_part_number, component_serial_number, component_name, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mr10, "2026-07-01", ac6, "27", "GL-2026-002",
		"Rigging check — aileron and airbrake travel", "inspection", "airframe",
		90, org1, "hangar", "LS4-CB-002", "CB-2022-008", "Airbrake control cable", userID)

	c1 := db.GenerateID()
	c2 := db.GenerateID()
	c3 := db.GenerateID()
	c4 := db.GenerateID()
	c5 := db.GenerateID()
	db.DB.Exec("INSERT INTO certifications (id, name, type, category, issue_date, expiry_date, authority, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", c1, "Part-66 B1.1", "license", "B1.1", "2020-03-01", "2027-03-01", "EASA", userID)
	db.DB.Exec("INSERT INTO certifications (id, name, type, category, aircraft_type, issue_date, expiry_date, authority, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", c2, "Type Rating A320 (CFM56)", "type_rating", "B1.1", "A320", "2021-06-15", "2027-06-15", "EASA", userID)
	db.DB.Exec("INSERT INTO certifications (id, name, type, category, aircraft_type, issue_date, expiry_date, authority, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", c3, "Type Rating Cessna 172", "type_rating", "B1.1", "C172", "2020-03-01", "2027-03-01", "EASA", userID)
	db.DB.Exec("INSERT INTO certifications (id, name, type, category, issue_date, expiry_date, authority, scope, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", c4, "Company Authorization — CAT A", "company_authorization", "B1.1", "2022-01-01", "2027-01-01", "EASA", "Line replacement and ELT checks", userID)
	db.DB.Exec("INSERT INTO certifications (id, name, type, issue_date, expiry_date, authority, user_id) VALUES (?, ?, ?, ?, ?, ?, ?)", c5, "EWIS Training", "training", "2025-06-01", "2028-06-01", "EASA", userID)
}

func SignOut(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		db.DB.Exec("DELETE FROM sessions WHERE id = ?", cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: "session_id", Value: "", MaxAge: -1, Path: "/", SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/auth/signin", http.StatusSeeOther)
}

func createSession(w http.ResponseWriter, userID string) {
	sessionID := db.GenerateID()
	_, err := db.DB.Exec(
		"INSERT INTO sessions (id, user_id) VALUES (?, ?)",
		sessionID, userID,
	)
	if err != nil {
		log.Printf("Create session error: %v", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("SESSION_SECRET") != "",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})
}
