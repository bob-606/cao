package handlers

import (
	"net/http"

	"logbook/db"
	"logbook/middleware"
)

func ReportsIndex(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var totals struct {
		Records                   int
		TotalTime, SupervisedTime int64
	}
	db.DB.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(total_time),0), COALESCE(SUM(supervised_time),0)
		FROM maintenance_records WHERE user_id = ?
	`, userID).Scan(&totals.Records, &totals.TotalTime, &totals.SupervisedTime)

	type breakdown struct {
		Name  string
		Count int
		Time  int64
	}

	var byAircraft []breakdown
	rows, _ := db.DB.Query(`
		SELECT a.registration, COUNT(*), COALESCE(SUM(m.total_time),0)
		FROM maintenance_records m JOIN aircraft a ON a.id = m.aircraft_id
		WHERE m.user_id = ? GROUP BY m.aircraft_id ORDER BY SUM(m.total_time) DESC
	`, userID)
	if rows != nil {
		for rows.Next() {
			var b breakdown
			rows.Scan(&b.Name, &b.Count, &b.Time)
			byAircraft = append(byAircraft, b)
		}
		rows.Close()
	}

	var byWorkType []breakdown
	rows, _ = db.DB.Query(`
		SELECT work_type, COUNT(*), COALESCE(SUM(total_time),0)
		FROM maintenance_records WHERE user_id = ?
		GROUP BY work_type ORDER BY SUM(total_time) DESC
	`, userID)
	if rows != nil {
		for rows.Next() {
			var b breakdown
			rows.Scan(&b.Name, &b.Count, &b.Time)
			byWorkType = append(byWorkType, b)
		}
		rows.Close()
	}

	var byATA []breakdown
	rows, _ = db.DB.Query(`
		SELECT COALESCE(m.ata_chapter, 'N/A'), COUNT(*), COALESCE(SUM(m.total_time),0)
		FROM maintenance_records m WHERE m.user_id = ?
		GROUP BY m.ata_chapter ORDER BY SUM(m.total_time) DESC LIMIT 15
	`, userID)
	if rows != nil {
		for rows.Next() {
			var b breakdown
			rows.Scan(&b.Name, &b.Count, &b.Time)
			byATA = append(byATA, b)
		}
		rows.Close()
	}

	var byCategory []breakdown
	rows, _ = db.DB.Query(`
		SELECT maintenance_category, COUNT(*), COALESCE(SUM(total_time),0)
		FROM maintenance_records WHERE user_id = ?
		GROUP BY maintenance_category ORDER BY SUM(total_time) DESC
	`, userID)
	if rows != nil {
		for rows.Next() {
			var b breakdown
			rows.Scan(&b.Name, &b.Count, &b.Time)
			byCategory = append(byCategory, b)
		}
		rows.Close()
	}

	var byMonth []breakdown
	rows, _ = db.DB.Query(`
		SELECT substr(date,1,7) as month, COUNT(*), COALESCE(SUM(total_time),0)
		FROM maintenance_records WHERE user_id = ?
		GROUP BY month ORDER BY month DESC LIMIT 12
	`, userID)
	if rows != nil {
		for rows.Next() {
			var b breakdown
			rows.Scan(&b.Name, &b.Count, &b.Time)
			byMonth = append(byMonth, b)
		}
		rows.Close()
	}

	var byOrg []breakdown
	rows, _ = db.DB.Query(`
		SELECT COALESCE(o.name, 'N/A'), COUNT(*), COALESCE(SUM(m.total_time),0)
		FROM maintenance_records m LEFT JOIN organizations o ON o.id = m.organization_id
		WHERE m.user_id = ? GROUP BY m.organization_id ORDER BY SUM(m.total_time) DESC
	`, userID)
	if rows != nil {
		for rows.Next() {
			var b breakdown
			rows.Scan(&b.Name, &b.Count, &b.Time)
			byOrg = append(byOrg, b)
		}
		rows.Close()
	}

	supervisionRatio := 0.0
	if totals.TotalTime > 0 {
		supervisionRatio = float64(totals.SupervisedTime) / float64(totals.TotalTime) * 100
	}

	render(w, r, "reports/index", map[string]interface{}{
		"TotalRecords":     totals.Records,
		"TotalTime":        db.FormatMinutes(totals.TotalTime),
		"SupervisedTime":   db.FormatMinutes(totals.SupervisedTime),
		"SupervisionRatio": supervisionRatio,
		"ByAircraft":       byAircraft,
		"ByWorkType":       byWorkType,
		"ByATA":            byATA,
		"ByCategory":       byCategory,
		"ByMonth":          byMonth,
		"ByOrganization":   byOrg,
	})
}
