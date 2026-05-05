// Package api provides HTTP handlers for the Gardener application's REST API.
//
// It implements endpoints for plant record management:
//   - GET /api/plants: retrieve all plant records
//   - POST /api/plants: create or update a plant record (supports multipart form and JSON)
//   - DELETE /api/plants?serial=<id>: remove a plant record and associated photos
//   - GET /api/photos/<serial>/<filename>: serve plant photo files
//
// All handlers follow standard HTTP semantics and return appropriate status codes:
//   - 200 OK: successful read or delete
//   - 201 Created: successful create/update
//   - 400 Bad Request: invalid input or missing parameters
//   - 404 Not Found: requested resource does not exist
//   - 409 Conflict: duplicate serial number on create
//   - 500 Internal Server Error: database or file system errors
package api

import (
	"encoding/json"
	"gardener/internal/db"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// GetPlants handles GET /api/plants requests.
//
// It retrieves all plant records from the database via db.ListPlants()
// and returns them as a JSON array. If the database query fails, it returns
// a 500 Internal Server Error with the error message.
//
// Response format:
//
//	[
//	  {
//	    "id": 1,
//	    "serial_number": "PKG-001",
//	    "plant_type": "Tomato",
//	    ...
//	  }
//	]
//
// The Content-Type header is set to application/json.
func GetPlants(w http.ResponseWriter, r *http.Request) {
	plants, err := db.ListPlants()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(plants); err != nil {
		log.Printf("⚠️ Failed to encode plants response: %v", err)
	}
}

// SavePlant handles POST /api/plants requests for creating or updating plant records.
//
// It supports two request formats:
//   - multipart/form-data: for file uploads (photos) with form fields
//   - application/json: for programmatic API clients
//
// For multipart requests, the handler:
//   - Parses form fields into a db.Plant struct
//   - Processes photo uploads via db.SavePhotoFile(), generating unique filenames
//   - Handles photo deletion requests via the photosToRemove field
//   - Updates the photos JSON array stored in the database
//
// For JSON requests, it decodes the request body directly into db.Plant.
//
// Validation:
//   - serial_number is required and must match ^[A-Za-z0-9\-_]{1,}$
//   - serial_number must be unique (checked before INSERT for new records)
//   - seed_remainder is clamped to non-negative values
//
// Persistence:
//   - If p.ID > 0: performs UPDATE by ID (editing existing record)
//   - If p.ID == 0: performs INSERT with uniqueness check (creating new record)
//
// Response:
//   - 201 Created with the saved plant as JSON on success
//   - 400 Bad Request for validation errors
//   - 409 Conflict for duplicate serial_number
//   - 500 Internal Server Error for database or file operation failures
func SavePlant(w http.ResponseWriter, r *http.Request) {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	var p db.Plant
	var originalSerial string

	if mediaType == "multipart/form-data" {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "parse error", http.StatusBadRequest)
			return
		}

		originalSerial = r.FormValue("_original_serial")
		p.ID, _ = strconv.Atoi(r.FormValue("id"))
		p.SerialNumber = strings.ToUpper(strings.TrimSpace(r.FormValue("serial_number")))
		p.PlantType = r.FormValue("plant_type")
		p.SeedSource = r.FormValue("seed_source")
		p.Description = r.FormValue("description")
		p.FruitDesc = r.FormValue("fruit_description")
		p.Technique = r.FormValue("technique")
		p.PlantingYears = r.FormValue("planting_years")
		p.Comments = r.FormValue("comments")
		p.Tags = r.FormValue("tags")
		p.SeedRemainder, _ = strconv.Atoi(r.FormValue("seed_remainder"))
		if p.SeedRemainder < 0 {
			p.SeedRemainder = 0
		}

		// Load existing photos array from form data.
		photos := []string{}
		if photosRaw := r.FormValue("photos"); photosRaw != "" {
			_ = json.Unmarshal([]byte(photosRaw), &photos)
		}

		// Process photo deletion requests.
		photosToRemoveRaw := r.FormValue("photosToRemove")
		var photosToRemove []string
		if photosToRemoveRaw != "" {
			_ = json.Unmarshal([]byte(photosToRemoveRaw), &photosToRemove)
		}

		// Remove marked photos from array and delete files from disk.
		for _, filename := range photosToRemove {
			newPhotos := []string{}
			for _, p := range photos {
				if p != filename {
					newPhotos = append(newPhotos, p)
				}
			}
			photos = newPhotos

			photoPath := filepath.Join(db.GetPhotoDir(p.SerialNumber), filename)
			if err := os.Remove(photoPath); err != nil && !os.IsNotExist(err) {
				log.Printf("⚠️ Failed to delete photo %s: %v", filename, err)
			}

			ext := filepath.Ext(filename)
			thumbName := strings.TrimSuffix(filename, ext) + "_thumb" + ext
			thumbPath := filepath.Join(db.GetPhotoDir(p.SerialNumber), thumbName)
			if err := os.Remove(thumbPath); err != nil && !os.IsNotExist(err) {
				log.Printf("⚠️ Failed to delete thumbnail %s: %v", thumbName, err)
			}
		}

		// Process new photo upload if present.
		file, header, err := r.FormFile("photo")
		if err == nil && file != nil && header.Filename != "" {
			defer func() {
				if cerr := file.Close(); cerr != nil {
					log.Printf("⚠️ Failed to close uploaded file: %v", cerr)
				}
			}()
			savedName, err := db.SavePhotoFile(p.SerialNumber, file, header.Filename)
			if err != nil {
				log.Printf("⚠️ Photo error: %v", err)
			} else if savedName != "" {
				found := false
				for _, existing := range photos {
					if existing == savedName {
						found = true
						break
					}
				}
				if !found {
					photos = append(photos, savedName)
				}
			}
		}

		// Save updated photos array as JSON string.
		photosJSON, _ := json.Marshal(photos)
		p.Photos = string(photosJSON)

	} else {
		// Handle JSON request body.
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		p.SerialNumber = strings.ToUpper(strings.TrimSpace(p.SerialNumber))
		originalSerial = p.SerialNumber
	}

	// Validate serial_number format and presence.
	if p.SerialNumber == "" {
		http.Error(w, "serial_number is required", http.StatusBadRequest)
		return
	}
	if !regexp.MustCompile(`^[A-Za-z0-9\-_]{1,}$`).MatchString(p.SerialNumber) {
		http.Error(w, "serial_number: only letters, numbers, dashes or underscores", http.StatusBadRequest)
		return
	}

	// Check uniqueness only for new records or when serial_number changes.
	needsCheck := originalSerial == "" || originalSerial != p.SerialNumber
	if needsCheck {
		var count int
		err := db.DB.QueryRow("SELECT COUNT(*) FROM plants WHERE serial_number = ?", p.SerialNumber).Scan(&count)
		if err == nil && count > 0 {
			http.Error(w, "Запись с таким № наклейки уже существует", http.StatusConflict)
			return
		}
	}

	// Persist to database: UPDATE by ID or INSERT new record.
	var err error
	if p.ID > 0 {
		_, err = db.DB.Exec(`UPDATE plants SET 
			plant_type=?, seed_source=?, description=?, fruit_description=?, 
			technique=?, planting_years=?, comments=?, seed_remainder=?, 
			photo_path=?, tags=?, photos=?
			WHERE id=?`,
			p.PlantType, p.SeedSource, p.Description, p.FruitDesc,
			p.Technique, p.PlantingYears, p.Comments, p.SeedRemainder,
			p.PhotoPath, p.Tags, p.Photos, p.ID)
	} else {
		_, err = db.DB.Exec(`INSERT INTO plants 
			(serial_number, plant_type, seed_source, description, fruit_description, 
			 technique, planting_years, comments, seed_remainder, photo_path, tags, photos)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			p.SerialNumber, p.PlantType, p.SeedSource, p.Description,
			p.FruitDesc, p.Technique, p.PlantingYears, p.Comments,
			p.SeedRemainder, p.PhotoPath, p.Tags, p.Photos)
	}

	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "UNIQUE constraint") {
			http.Error(w, "Запись с таким № наклейки уже существует", http.StatusConflict)
			return
		}
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.Printf("⚠️ Failed to encode plant response: %v", err)
	}
}

// DeletePlant handles DELETE /api/plants?serial=<id> requests.
//
// It removes the plant record from the database via db.DeletePlant()
// and deletes associated photo files via db.DeletePhotos().
//
// Parameters:
//   - serial: query parameter specifying the plant's unique serial number
//
// Response:
//   - 200 OK on successful deletion (idempotent: also returns 200 if record did not exist)
//   - 400 Bad Request if serial parameter is missing
//   - 500 Internal Server Error if database deletion fails
//
// Note: Photo deletion errors are logged but do not cause the handler to fail,
// ensuring the database record is removed even if file cleanup encounters issues.
func DeletePlant(w http.ResponseWriter, r *http.Request) {
	serial := r.URL.Query().Get("serial")
	if serial == "" {
		http.Error(w, "serial parameter required", http.StatusBadRequest)
		return
	}
	if err := db.DeletePlant(serial); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.DeletePhotos(serial); err != nil {
		log.Printf("⚠️ Failed to delete photos for %s: %v", serial, err)
	}
	w.WriteHeader(http.StatusOK)
}

// GetPhoto handles GET /api/photos/<serial>/<filename> requests.
//
// It serves static photo files stored in the application's photo directory.
// The handler performs security checks to prevent path traversal attacks:
//   - Validates that the URL path contains exactly two segments (serial and filename)
//   - Rejects requests containing ".." in either segment
//
// Parameters:
//   - serial: the plant's unique serial number (used to locate the photo directory)
//   - filename: the name of the photo file to serve
//
// Response:
//   - 200 OK with the image file content on success
//   - 400 Bad Request for malformed paths
//   - 403 Forbidden for path traversal attempts
//   - 404 Not Found if the file does not exist
func GetPhoto(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/photos/"), "/")
	if len(parts) != 2 {
		http.Error(w, "invalid", http.StatusBadRequest)
		return
	}
	serial, filename := parts[0], parts[1]
	if strings.Contains(filename, "..") || strings.Contains(serial, "..") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	path := filepath.Join(db.GetPhotoDir(serial), filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, path)
}
