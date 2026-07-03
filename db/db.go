package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) {
	if dbPath == "" {
		dbPath = "./data/cao.db"
	}

	dir := dbPath
	for i := len(dbPath) - 1; i >= 0; i-- {
		if dbPath[i] == '/' || dbPath[i] == '\\' {
			dir = dbPath[:i]
			break
		}
	}
	if dir != dbPath {
		os.MkdirAll(dir, 0755)
	}

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	DB.SetMaxOpenConns(1)

	if err := DB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	createSchema()
	runMigrations()
	log.Println("CAO database initialized successfully")
}

func createSchema() {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS organizations (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'camo',
		easa_part_number TEXT,
		address TEXT,
		country TEXT DEFAULT 'EE',
		owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS aircraft (
		id TEXT PRIMARY KEY,
		registration TEXT NOT NULL,
		type TEXT NOT NULL,
		variant TEXT,
		model TEXT,
		serial_number TEXT,
		year_of_manufacture INTEGER,
		category TEXT DEFAULT 'airplane',
		engine_type TEXT,
		apu_type TEXT,
		prop_type TEXT,
		mtow INTEGER,
		max_pax INTEGER,
		owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE(registration, owner_id)
	);

	CREATE TABLE IF NOT EXISTS maintenance_records (
		id TEXT PRIMARY KEY,
		date TEXT NOT NULL,
		aircraft_id TEXT NOT NULL REFERENCES aircraft(id),
		ata_chapter TEXT,
		work_order TEXT,
		task_card TEXT,
		defect_ref TEXT,
		description TEXT NOT NULL,
		work_type TEXT DEFAULT 'scheduled',
		maintenance_category TEXT DEFAULT 'airframe',
		total_time INTEGER NOT NULL,
		supervised_time INTEGER DEFAULT 0,
		supervisor_name TEXT,
		supervisor_cert TEXT,
		certifying_staff_name TEXT,
		certifying_staff_cert TEXT,
		organization_id TEXT REFERENCES organizations(id),
		location TEXT,
		aircraft_tsn INTEGER,
		aircraft_csn INTEGER,
		component_part_number TEXT,
		component_serial_number TEXT,
		component_name TEXT,
		remarks TEXT,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS certifications (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT DEFAULT 'license',
		category TEXT,
		aircraft_type TEXT,
		issue_date TEXT,
		expiry_date TEXT,
		authority TEXT DEFAULT 'EASA',
		scope TEXT,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS supervised_work (
		id TEXT PRIMARY KEY,
		record_id TEXT NOT NULL REFERENCES maintenance_records(id) ON DELETE CASCADE,
		supervisor_name TEXT NOT NULL,
		supervisor_cert TEXT,
		task_description TEXT,
		supervised_minutes INTEGER DEFAULT 0,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS ata_chapters (
		chapter TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		section TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_maintenance_user ON maintenance_records(user_id);
	CREATE INDEX IF NOT EXISTS idx_maintenance_date ON maintenance_records(date);
	CREATE INDEX IF NOT EXISTS idx_maintenance_aircraft ON maintenance_records(aircraft_id);
	CREATE INDEX IF NOT EXISTS idx_maintenance_org ON maintenance_records(organization_id);
	CREATE INDEX IF NOT EXISTS idx_aircraft_owner ON aircraft(owner_id);
	CREATE INDEX IF NOT EXISTS idx_org_owner ON organizations(owner_id);
	CREATE INDEX IF NOT EXISTS idx_certs_user ON certifications(user_id);
	CREATE INDEX IF NOT EXISTS idx_supervised_record ON supervised_work(record_id);

	CREATE TABLE IF NOT EXISTS oauth_accounts (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		provider TEXT NOT NULL,
		provider_user_id TEXT NOT NULL,
		UNIQUE(provider, provider_user_id)
	);

	CREATE TABLE IF NOT EXISTS magic_links (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL,
		token TEXT NOT NULL UNIQUE,
		used INTEGER DEFAULT 0,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := DB.Exec(schema); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	seedATAChapters()
}

func seedATAChapters() {
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM ata_chapters").Scan(&count)
	if count > 0 {
		return
	}

	chapters := []struct{ ch, name, section string }{
		{"21", "Air Conditioning", "ATA21"},
		{"22", "Auto Flight", "ATA22"},
		{"23", "Communications", "ATA23"},
		{"24", "Electrical Power", "ATA24"},
		{"25", "Equipment / Furnishings", "ATA25"},
		{"26", "Fire Protection", "ATA26"},
		{"27", "Flight Controls", "ATA27"},
		{"28", "Fuel", "ATA28"},
		{"29", "Hydraulic Power", "ATA29"},
		{"30", "Ice and Rain Protection", "ATA30"},
		{"31", "Indicating / Recording Systems", "ATA31"},
		{"32", "Landing Gear", "ATA32"},
		{"33", "Lights", "ATA33"},
		{"34", "Navigation", "ATA34"},
		{"35", "Oxygen", "ATA35"},
		{"36", "Pneumatic", "ATA36"},
		{"38", "Water / Waste", "ATA38"},
		{"49", "Airborne Auxiliary Power", "ATA49"},
		{"51", "Standard Practices / Structures", "ATA51"},
		{"52", "Doors", "ATA52"},
		{"53", "Fuselage", "ATA53"},
		{"54", "Nacelles / Pylons", "ATA54"},
		{"55", "Stabilizers", "ATA55"},
		{"56", "Windows", "ATA56"},
		{"57", "Wings", "ATA57"},
		{"61", "Propellers", "ATA61"},
		{"63", "Rotor Drives", "ATA63"},
		{"65", "Tail Rotor", "ATA65"},
		{"67", "Rotors Flight Control", "ATA67"},
		{"71", "Power Plant", "ATA71"},
		{"72", "Engine (Turbine/Turboprop)", "ATA72"},
		{"73", "Engine Fuel and Control", "ATA73"},
		{"74", "Ignition", "ATA74"},
		{"75", "Engine Air", "ATA75"},
		{"76", "Engine Controls", "ATA76"},
		{"77", "Engine Indicating", "ATA77"},
		{"78", "Exhaust", "ATA78"},
		{"79", "Engine Oil", "ATA79"},
		{"80", "Starting", "ATA80"},
		{"82", "Water Injection", "ATA82"},
		{"91", "Charts", "ATA91"},
	}

	for _, c := range chapters {
		DB.Exec("INSERT OR IGNORE INTO ata_chapters (chapter, name, section) VALUES (?, ?, ?)", c.ch, c.name, c.section)
	}
}

func runMigrations() {
	migrations := []struct {
		check string
		sql   string
	}{
		{
			check: "SELECT COUNT(*) FROM pragma_table_info('aircraft') WHERE name = 'wing_span_m'",
			sql:   "ALTER TABLE aircraft ADD COLUMN wing_span_m REAL",
		},
		{
			check: "SELECT COUNT(*) FROM pragma_table_info('aircraft') WHERE name = 'empty_weight_kg'",
			sql:   "ALTER TABLE aircraft ADD COLUMN empty_weight_kg INTEGER",
		},
	}

	for _, m := range migrations {
		var count int
		DB.QueryRow(m.check).Scan(&count)
		if count == 0 {
			if _, err := DB.Exec(m.sql); err != nil {
				log.Printf("Migration warning: %v", err)
			}
		}
	}
}

func GenerateID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	f, _ := os.Open("/dev/urandom")
	f.Read(b)
	f.Close()
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}

func FormatMinutes(m int64) string {
	hours := m / 60
	mins := m % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
