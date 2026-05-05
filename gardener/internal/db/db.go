// Package db provides database operations for the Gardener application.
// It manages SQLite persistence for plant records, including creation,
// retrieval, update, and deletion of plant data with associated metadata.
//
// The database is stored in the user's home directory at ~/.gardener/plants.db.
// Connection uses WAL journal mode and a 5-second busy timeout for concurrency safety.
package db

import (
	"database/sql"
	"log"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
)

// DB is the global database connection handle.
// It is initialized by Init() and used by all repository functions.
// The connection is safe for concurrent use by multiple goroutines.
var DB *sql.DB

// Plant represents a plant record in the application.
//
// Fields are tagged for JSON serialization/deserialization.
// The Photos field contains a JSON-encoded array of photo filenames.
// SeedRemainder is validated to be non-negative at the database level.
type Plant struct {
	ID            int    `json:"id"`
	SerialNumber  string `json:"serial_number"`
	PlantType     string `json:"plant_type"`
	SeedSource    string `json:"seed_source"`
	Description   string `json:"description"`
	FruitDesc     string `json:"fruit_description"`
	Technique     string `json:"technique"`
	PlantingYears string `json:"planting_years"`
	Comments      string `json:"comments"`
	SeedRemainder int    `json:"seed_remainder"`
	PhotoPath     string `json:"photo_path"`
	Tags          string `json:"tags"`
	Photos        string `json:"photos"`
}

// Init initializes the SQLite database connection and schema.
//
// It performs the following steps:
//   - Creates the ~/.gardener directory if it does not exist
//   - Opens a connection to plants.db with WAL mode and busy_timeout=5000
//   - Creates the plants table if it does not exist
//   - Runs a migration to add the photos column if missing
//
// The plants table schema enforces:
//   - serial_number as UNIQUE NOT NULL
//   - seed_remainder >= 0 via CHECK constraint
//   - Default empty strings for text fields, 0 for integers
//   - created_at timestamp defaulting to CURRENT_TIMESTAMP
//
// Returns an error if directory creation, database connection,
// or schema initialization fails.
func Init() error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".gardener")
	os.MkdirAll(dir, 0755)

	dbPath := filepath.Join(dir, "plants.db")
	var err error
	DB, err = sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}

	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS plants (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		serial_number TEXT UNIQUE NOT NULL,
		plant_type TEXT DEFAULT '', seed_source TEXT DEFAULT '',
		description TEXT DEFAULT '', fruit_description TEXT DEFAULT '',
		technique TEXT DEFAULT '', planting_years TEXT DEFAULT '',
		comments TEXT DEFAULT '', seed_remainder INTEGER DEFAULT 0 CHECK(seed_remainder >= 0),
		photo_path TEXT DEFAULT '', tags TEXT DEFAULT '',
		photos TEXT DEFAULT '[]',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return err
	}

	// Migration: add photos column if it does not exist.
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('plants') WHERE name='photos'`).Scan(&count); err != nil {
		log.Printf("⚠️ Failed to check photos column: %v", err)
		return err
	}
	if count == 0 {
		_, _ = DB.Exec("ALTER TABLE plants ADD COLUMN photos TEXT DEFAULT '[]'")
		log.Printf("✅ Migrated: added column photos")
	}
	return nil
}

// ListPlants retrieves all plant records from the database.
//
// Results are ordered by plant_type in ascending order.
// Filtering, searching, and highlighting are performed on the client side.
//
// Returns:
//   - A slice of Plant structs containing all records
//   - An error if the query execution or row scanning fails
func ListPlants() ([]Plant, error) {
	rows, err := DB.Query("SELECT id, serial_number, plant_type, seed_source, description, fruit_description, technique, planting_years, comments, seed_remainder, photo_path, tags, photos FROM plants ORDER BY plant_type")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plants []Plant
	for rows.Next() {
		var p Plant
		if err := rows.Scan(&p.ID, &p.SerialNumber, &p.PlantType, &p.SeedSource, &p.Description, &p.FruitDesc, &p.Technique, &p.PlantingYears, &p.Comments, &p.SeedRemainder, &p.PhotoPath, &p.Tags, &p.Photos); err != nil {
			return nil, err
		}
		plants = append(plants, p)
	}
	return plants, rows.Err()
}

// SavePlant inserts a new plant record into the database.
//
// The function enforces:
//   - serial_number uniqueness via UNIQUE constraint (caller must check before insert)
//   - seed_remainder non-negativity by clamping negative values to 0
//
// For updating existing records, the caller should use an UPDATE query
// with the plant's ID, as this function only performs INSERT operations.
//
// Parameters:
//   - p: pointer to a Plant struct with fields populated
//
// Returns:
//   - An error if the INSERT fails (e.g., duplicate serial_number, constraint violation)
//   - nil on successful insertion
func SavePlant(p *Plant) error {
	if p.SeedRemainder < 0 {
		p.SeedRemainder = 0
	}

	_, err := DB.Exec(`INSERT INTO plants 
		(serial_number, plant_type, seed_source, description, fruit_description, technique, planting_years, comments, seed_remainder, photo_path, tags, photos)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.SerialNumber, p.PlantType, p.SeedSource, p.Description, p.FruitDesc, p.Technique, p.PlantingYears, p.Comments, p.SeedRemainder, p.PhotoPath, p.Tags, p.Photos)

	return err
}

// DeletePlant removes a plant record from the database by its serial number.
//
// This operation is irreversible and does not cascade to associated photo files.
// Callers should invoke DeletePhotos() separately to clean up the file system.
//
// Parameters:
//   - serial: the unique serial number of the plant to delete
//
// Returns:
//   - An error if the DELETE query fails
//   - nil if the record was deleted or did not exist (idempotent behavior)
func DeletePlant(serial string) error {
	_, err := DB.Exec("DELETE FROM plants WHERE serial_number = ?", serial)
	return err
}
