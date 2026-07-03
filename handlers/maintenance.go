package handlers

import (
	"database/sql"
	"log"
	"net/http"

	"logbook/db"
	"logbook/middleware"
)

func MaintenanceIndex(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	q := r.URL.Query()

	query := `SELECT m.id, m.date, m.description, m.work_type, m.maintenance_category,
		m.total_time, m.supervised_time, m.ata_chapter, m.location,
		a.registration, a.type, o.name, o.type
		FROM maintenance_records m
		JOIN aircraft a ON a.id = m.aircraft_id
		LEFT JOIN organizations o ON o.id = m.organization_id
		WHERE m.user_id = ?`
	args := []interface{}{userID}

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
	if cat := q.Get("maintenanceCategory"); cat != "" {
		query += " AND m.maintenance_category = ?"
		args = append(args, cat)
	}
	if ata := q.Get("ataChapter"); ata != "" {
		query += " AND m.ata_chapter = ?"
		args = append(args, ata)
	}
	if search := q.Get("q"); search != "" {
		query += " AND (m.description LIKE ? OR m.work_order LIKE ? OR m.task_card LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}

	query += " ORDER BY m.date DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Query error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type RecordRow struct {
		ID                   string
		Date                 string
		Description          string
		WorkType             string
		MaintenanceCategory  string
		TotalTime            int64
		SupervisedTime       int64
		ATAChapter           *string
		Location             *string
		Aircraft             string
		AircraftType         string
		OrganizationName     *string
		OrganizationType     *string
	}

	var records []RecordRow
	var totalTime, totalSupervised int64
	for rows.Next() {
		var r RecordRow
		if err := rows.Scan(&r.ID, &r.Date, &r.Description, &r.WorkType, &r.MaintenanceCategory,
			&r.TotalTime, &r.SupervisedTime, &r.ATAChapter, &r.Location,
			&r.Aircraft, &r.AircraftType, &r.OrganizationName, &r.OrganizationType); err != nil {
			continue
		}
		totalTime += r.TotalTime
		totalSupervised += r.SupervisedTime
		records = append(records, r)
	}

	aircraftRows, _ := db.DB.Query("SELECT id, registration, type FROM aircraft WHERE owner_id = ? ORDER BY registration", userID)
	defer aircraftRows.Close()
	type ac struct{ ID, Registration, Type string }
	var aircraft []ac
	for aircraftRows.Next() {
		var a ac
		aircraftRows.Scan(&a.ID, &a.Registration, &a.Type)
		aircraft = append(aircraft, a)
	}

	orgRows, _ := db.DB.Query("SELECT id, name, type FROM organizations WHERE owner_id = ? ORDER BY name", userID)
	defer orgRows.Close()
	type org struct{ ID, Name, Type string }
	var organizations []org
	for orgRows.Next() {
		var o org
		orgRows.Scan(&o.ID, &o.Name, &o.Type)
		organizations = append(organizations, o)
	}

	ataRows, _ := db.DB.Query("SELECT chapter, name FROM ata_chapters ORDER BY chapter")
	defer ataRows.Close()
	type ataRow struct{ Chapter, Name string }
	var ataChapters []ataRow
	for ataRows.Next() {
		var a ataRow
		ataRows.Scan(&a.Chapter, &a.Name)
		ataChapters = append(ataChapters, a)
	}

	qv := make(map[string]string)
	for k, v := range q {
		if len(v) > 0 {
			qv[k] = v[0]
		}
	}

	render(w, r, "maintenance/index", map[string]interface{}{
		"Records":        records,
		"TotalTime":      db.FormatMinutes(totalTime),
		"TotalSupervised": db.FormatMinutes(totalSupervised),
		"Aircraft":       aircraft,
		"Organizations":  organizations,
		"ATAChapters":    ataChapters,
		"Query":          q,
		"QueryMap":       qv,
		"HasFilters":     q.Get("dateFrom") != "" || q.Get("dateTo") != "" || q.Get("aircraftId") != "" || q.Get("organizationId") != "" || q.Get("workType") != "" || q.Get("maintenanceCategory") != "" || q.Get("ataChapter") != "" || q.Get("q") != "",
	})
}

func MaintenanceNew(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	aircraftRows, _ := db.DB.Query("SELECT id, registration, type, category FROM aircraft WHERE owner_id = ? ORDER BY registration", userID)
	defer aircraftRows.Close()
	type ac struct{ ID, Registration, Type, Category string }
	var aircraft []ac
	for aircraftRows.Next() {
		var a ac
		aircraftRows.Scan(&a.ID, &a.Registration, &a.Type, &a.Category)
		aircraft = append(aircraft, a)
	}

	orgRows, _ := db.DB.Query("SELECT id, name, type FROM organizations WHERE owner_id = ? ORDER BY name", userID)
	defer orgRows.Close()
	type org struct{ ID, Name, Type string }
	var organizations []org
	for orgRows.Next() {
		var o org
		orgRows.Scan(&o.ID, &o.Name, &o.Type)
		organizations = append(organizations, o)
	}

	ataRows, _ := db.DB.Query("SELECT chapter, name FROM ata_chapters ORDER BY chapter")
	defer ataRows.Close()
	type ataRow struct{ Chapter, Name string }
	var ataChapters []ataRow
	for ataRows.Next() {
		var a ataRow
		ataRows.Scan(&a.Chapter, &a.Name)
		ataChapters = append(ataChapters, a)
	}

	render(w, r, "maintenance/new", map[string]interface{}{
		"Aircraft":      aircraft,
		"Organizations": organizations,
		"ATAChapters":   ataChapters,
	})
}

func MaintenanceCreate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	id := db.GenerateID()
	date := r.FormValue("date")
	aircraftID := r.FormValue("aircraftId")
	ataChapter := r.FormValue("ataChapter")
	workOrder := r.FormValue("workOrder")
	taskCard := r.FormValue("taskCard")
	defectRef := r.FormValue("defectRef")
	description := r.FormValue("description")
	workType := r.FormValue("workType")
	cat := r.FormValue("maintenanceCategory")
	total := parseInt(r.FormValue("totalTime"))
	supervised := parseInt(r.FormValue("supervisedTime"))
	supervisorName := r.FormValue("supervisorName")
	supervisorCert := r.FormValue("supervisorCert")
	certName := r.FormValue("certifyingStaffName")
	certCert := r.FormValue("certifyingStaffCert")
	orgID := r.FormValue("organizationId")
	location := r.FormValue("location")
	tsn := parseInt(r.FormValue("aircraftTsn"))
	csn := parseInt(r.FormValue("aircraftCsn"))
	partNumber := r.FormValue("componentPartNumber")
	serialNumber := r.FormValue("componentSerialNumber")
	compName := r.FormValue("componentName")
	remarks := r.FormValue("remarks")

	if date == "" || aircraftID == "" || description == "" || total == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var tsnp, csnp *int64
	if tsn > 0 {
		tsnp = &tsn
	}
	if csn > 0 {
		csnp = &csn
	}

	_, err := db.DB.Exec(`
		INSERT INTO maintenance_records (id, date, aircraft_id, ata_chapter, work_order, task_card,
			defect_ref, description, work_type, maintenance_category, total_time, supervised_time,
			supervisor_name, supervisor_cert, certifying_staff_name, certifying_staff_cert,
			organization_id, location, aircraft_tsn, aircraft_csn,
			component_part_number, component_serial_number, component_name, remarks, user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, date, aircraftID, nullIfEmpty(ataChapter), nullIfEmpty(workOrder), nullIfEmpty(taskCard),
		nullIfEmpty(defectRef), description, orDefault(workType, "scheduled"), orDefault(cat, "airframe"),
		total, supervised,
		nullIfEmpty(supervisorName), nullIfEmpty(supervisorCert),
		nullIfEmpty(certName), nullIfEmpty(certCert),
		nullIfEmpty(orgID), nullIfEmpty(location), tsnp, csnp,
		nullIfEmpty(partNumber), nullIfEmpty(serialNumber), nullIfEmpty(compName),
		nullIfEmpty(remarks), userID,
	)
	if err != nil {
		log.Printf("Insert maintenance error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if supervised > 0 && supervisorName != "" {
		supID := db.GenerateID()
		db.DB.Exec(`
			INSERT INTO supervised_work (id, record_id, supervisor_name, supervisor_cert, task_description, supervised_minutes, user_id)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			supID, id, supervisorName, nullIfEmpty(supervisorCert), nullIfEmpty(description), supervised, userID,
		)
	}

	http.Redirect(w, r, "/maintenance", http.StatusSeeOther)
}

func MaintenanceShow(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	id := r.PathValue("id")

	var m struct {
		Date, Description, WorkType, MaintenanceCategory string
		ATAChapter, WorkOrder, TaskCard, DefectRef        *string
		TotalTime, SupervisedTime                          int64
		SupervisorName, SupervisorCert                     *string
		CertName, CertCert                                 *string
		Location                                          *string
		AircraftTSN, AircraftCSN                           *int64
		PartNumber, SerialNumber, CompName                 *string
		Remarks                                            *string
		AircraftReg, AircraftType, Category                string
		OrgName, OrgType                                   *string
	}
	var ataCh, wo, tc, dr, supName, supCert, cName, cCert, loc string
	var rem string
	err := db.DB.QueryRow(`
		SELECT m.date, m.description, m.work_type, m.maintenance_category,
			m.ata_chapter, m.work_order, m.task_card, m.defect_ref,
			m.total_time, m.supervised_time,
			m.supervisor_name, m.supervisor_cert,
			m.certifying_staff_name, m.certifying_staff_cert,
			m.location, m.aircraft_tsn, m.aircraft_csn,
			m.component_part_number, m.component_serial_number, m.component_name,
			m.remarks,
			a.registration, a.type, a.category,
			o.name, o.type
		FROM maintenance_records m
		JOIN aircraft a ON a.id = m.aircraft_id
		LEFT JOIN organizations o ON o.id = m.organization_id
		WHERE m.id = ? AND m.user_id = ?`, id, userID,
	).Scan(&m.Date, &m.Description, &m.WorkType, &m.MaintenanceCategory,
		&ataCh, &wo, &tc, &dr,
		&m.TotalTime, &m.SupervisedTime,
		&supName, &supCert, &cName, &cCert,
		&loc, &m.AircraftTSN, &m.AircraftCSN,
		&m.PartNumber, &m.SerialNumber, &m.CompName,
		&rem,
		&m.AircraftReg, &m.AircraftType, &m.Category,
		&m.OrgName, &m.OrgType)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("Query error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if ataCh != "" {
		m.ATAChapter = &ataCh
	}
	if wo != "" {
		m.WorkOrder = &wo
	}
	if tc != "" {
		m.TaskCard = &tc
	}
	if dr != "" {
		m.DefectRef = &dr
	}
	if supName != "" {
		m.SupervisorName = &supName
	}
	if supCert != "" {
		m.SupervisorCert = &supCert
	}
	if cName != "" {
		m.CertName = &cName
	}
	if cCert != "" {
		m.CertCert = &cCert
	}
	if loc != "" {
		m.Location = &loc
	}
	if rem != "" {
		m.Remarks = &rem
	}

	// Get ATA chapter name
	var ataName string
	if m.ATAChapter != nil {
		db.DB.QueryRow("SELECT name FROM ata_chapters WHERE chapter = ?", *m.ATAChapter).Scan(&ataName)
	}

	render(w, r, "maintenance/show", map[string]interface{}{
		"Record":      m,
		"TotalTime":   db.FormatMinutes(m.TotalTime),
		"SupervisedTimeStr": db.FormatMinutes(m.SupervisedTime),
		"ATAName":     ataName,
		"RecordID":    id,
	})
}

func MaintenanceDelete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	id := r.PathValue("id")

	db.DB.Exec("DELETE FROM maintenance_records WHERE id = ? AND user_id = ?", id, userID)
	http.Redirect(w, r, "/maintenance", http.StatusSeeOther)
}
