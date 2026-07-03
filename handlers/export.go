package handlers

import (
	"log"
	"net/http"
	"time"

	"logbook/db"
	"logbook/middleware"

	"github.com/jung-kurt/gofpdf"
)

func ExportPDF(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	userName := middleware.GetUserName(r)

	records := getFilteredRecords(userID, r.URL.Query())
	exportPart66(w, userName, records)
}

type exportRow struct {
	Date, Description, WorkType, Category string
	ATAChapter, WorkOrder, TaskCard        string
	Aircraft, AircraftType                 string
	Organization                           string
	Location                               string
	Total, Supervised                      int64
	SupervisorName, CertName               string
	PartNumber, SerialNumber, CompName     string
	Tsn, Csn                               int64
}

func getFilteredRecords(userID string, q interface{ Get(string) string }) []exportRow {
	query := `SELECT m.date, m.description, m.work_type, m.maintenance_category,
		COALESCE(m.ata_chapter, ''), COALESCE(m.work_order, ''), COALESCE(m.task_card, ''),
		a.registration, a.type,
		COALESCE(o.name, ''),
		COALESCE(m.location, ''),
		m.total_time, m.supervised_time,
		COALESCE(m.supervisor_name, ''), COALESCE(m.certifying_staff_name, ''),
		COALESCE(m.component_part_number, ''), COALESCE(m.component_serial_number, ''), COALESCE(m.component_name, ''),
		COALESCE(m.aircraft_tsn, 0), COALESCE(m.aircraft_csn, 0)
		FROM maintenance_records m
		JOIN aircraft a ON a.id = m.aircraft_id
		LEFT JOIN organizations o ON o.id = m.organization_id
		WHERE m.user_id = ?`

	var args []interface{} = []interface{}{userID}

	if dateFrom := q.Get("dateFrom"); dateFrom != "" {
		query += " AND m.date >= ?"
		args = append(args, dateFrom)
	}
	if dateTo := q.Get("dateTo"); dateTo != "" {
		query += " AND m.date <= ?"
		args = append(args, dateTo+"T23:59:59")
	}
	if aircraftID := q.Get("aircraftId"); aircraftID != "" {
		query += " AND m.aircraft_id = ?"
		args = append(args, aircraftID)
	}
	if orgID := q.Get("organizationId"); orgID != "" {
		query += " AND m.organization_id = ?"
		args = append(args, orgID)
	}
	if workType := q.Get("workType"); workType != "" {
		query += " AND m.work_type = ?"
		args = append(args, workType)
	}
	if search := q.Get("q"); search != "" {
		like := "%" + search + "%"
		query += " AND (m.description LIKE ? OR m.work_order LIKE ?)"
		args = append(args, like, like)
	}

	query += " ORDER BY m.date DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Export query error: %v", err)
		return nil
	}
	defer rows.Close()

	var data []exportRow
	for rows.Next() {
		var r exportRow
		rows.Scan(&r.Date, &r.Description, &r.WorkType, &r.Category,
			&r.ATAChapter, &r.WorkOrder, &r.TaskCard,
			&r.Aircraft, &r.AircraftType,
			&r.Organization, &r.Location,
			&r.Total, &r.Supervised,
			&r.SupervisorName, &r.CertName,
			&r.PartNumber, &r.SerialNumber, &r.CompName,
			&r.Tsn, &r.Csn)
		data = append(data, r)
	}
	return data
}

// EASA Part-66 Logbook format
func exportPart66(w http.ResponseWriter, userName string, data []exportRow) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, "Part-66 Maintenance Logbook", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 5, "Engineer: "+userName+" | Generated: "+time.Now().Format("2 Jan 2006"), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	headers := []string{"Date", "A/C Reg", "Type", "ATA", "Work Type", "Category",
		"Description", "WO/Task", "Location", "Total", "Supervised", "Supervisor", "Cert Staff", "Part No", "Remarks"}
	colWidths := []float64{22, 18, 18, 10, 16, 16, 28, 20, 14, 12, 14, 16, 16, 18, 22}

	pdf.SetFont("Helvetica", "B", 5.5)
	pdf.SetFillColor(41, 41, 41)
	pdf.SetTextColor(255, 255, 255)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 5, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 5.5)
	pdf.SetTextColor(0, 0, 0)
	fill := false
	var sum struct{ total, supervised int64 }
	for _, d := range data {
		if fill {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		desc := truncate(d.Description, 22)
		wo := d.WorkOrder
		if d.TaskCard != "" {
			wo = d.TaskCard
		}
		vals := []string{
			d.Date, d.Aircraft, d.AircraftType, d.ATAChapter, d.WorkType, d.Category,
			desc, truncate(wo, 16), d.Location,
			fmtMin(d.Total), fmtMin(d.Supervised),
			truncate(d.SupervisorName, 14), truncate(d.CertName, 14),
			truncate(d.PartNumber, 14), truncate(d.Description, 18),
		}
		for i, v := range vals {
			pdf.CellFormat(colWidths[i], 4.5, v, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
		fill = !fill
		sum.total += d.Total
		sum.supervised += d.Supervised
	}

	pdf.Ln(3)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(0, 5, "Totals:", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	totals := []string{
		"Total Records: " + itoa(int64(len(data))),
		"Total Time: " + fmtMin(sum.total),
		"Supervised Time: " + fmtMin(sum.supervised),
	}
	if sum.total > 0 {
		ratio := float64(sum.supervised) / float64(sum.total) * 100
		totals = append(totals, "Supervision Ratio: "+itoa(int64(ratio))+"%")
	}
	for _, t := range totals {
		pdf.CellFormat(50, 4.5, t, "", 0, "L", false, 0, "")
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=logbook-part66.pdf")
	pdf.Output(w)
}
