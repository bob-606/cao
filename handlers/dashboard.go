package handlers

import (
	"net/http"

	"logbook/db"
	"logbook/middleware"
)

func Dashboard(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var totalRecords int
	var totalTime int64
	db.DB.QueryRow("SELECT COUNT(*), COALESCE(SUM(total_time), 0) FROM maintenance_records WHERE user_id = ?", userID).Scan(&totalRecords, &totalTime)

	var aircraftCount, orgCount, certCount int
	db.DB.QueryRow("SELECT COUNT(*) FROM aircraft WHERE owner_id = ?", userID).Scan(&aircraftCount)
	db.DB.QueryRow("SELECT COUNT(*) FROM organizations WHERE owner_id = ?", userID).Scan(&orgCount)
	db.DB.QueryRow("SELECT COUNT(*) FROM certifications WHERE user_id = ?", userID).Scan(&certCount)

	rows, err := db.DB.Query(`
		SELECT m.id, m.date, m.description, m.work_type, m.total_time,
			a.registration, o.name
		FROM maintenance_records m
		JOIN aircraft a ON a.id = m.aircraft_id
		LEFT JOIN organizations o ON o.id = m.organization_id
		WHERE m.user_id = ?
		ORDER BY m.date DESC LIMIT 5
	`, userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type RecordRow struct {
		ID          string
		Date        string
		Description string
		WorkType    string
		TotalTime   int64
		Aircraft    string
		OrgName     *string
	}

	var records []RecordRow
	for rows.Next() {
		var r RecordRow
		if err := rows.Scan(&r.ID, &r.Date, &r.Description, &r.WorkType, &r.TotalTime, &r.Aircraft, &r.OrgName); err != nil {
			continue
		}
		records = append(records, r)
	}

	render(w, r, "dashboard", map[string]interface{}{
		"TotalRecords":   totalRecords,
		"TotalTime":      db.FormatMinutes(totalTime),
		"AircraftCount":  aircraftCount,
		"OrgCount":       orgCount,
		"CertCount":      certCount,
		"RecentRecords":  records,
		"UserName":       middleware.GetUserName(r),
	})
}
