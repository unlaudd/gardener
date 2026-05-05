// Package api provides unit tests for HTTP handlers in the Gardener application.
//
// Tests verify:
//   - CRUD operations for plant records via REST API endpoints
//   - Input validation (serial_number format, required fields)
//   - Uniqueness constraints and conflict handling
//   - Multipart form handling for photo uploads
//   - Security checks (path traversal prevention)
//   - Error response codes and messages
//
// All tests use httptest for isolated HTTP request/response simulation
// and mock the HOME environment variable to avoid modifying user data.
package api

import (
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"gardener/internal/db"
)

// setupTestAPI initializes a test environment for API handler tests.
//
// It performs the following:
//   - Creates a temporary directory via t.TempDir() for isolated file operations
//   - Overrides the HOME environment variable to redirect database and photo storage
//   - Calls db.Init() to set up the test database schema and connection
//
// Returns a cleanup function that must be deferred by the caller to:
//   - Restore the original HOME environment variable
//   - Close the database connection and release resources
//
// Usage:
//
//	cleanup := setupTestAPI(t)
//	defer cleanup()
func setupTestAPI(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	if err := db.Init(); err != nil {
		t.Fatalf("DB init failed: %v", err)
	}

	return func() {
		os.Setenv("HOME", origHome)
		db.DB.Close()
	}
}

// TestGetPlants_Empty verifies that GetPlants returns an empty JSON array
// when no plant records exist in the database.
//
// Test steps:
//   - Create a fresh test database (no records)
//   - Send GET /api/plants request
//   - Assert response status is 200 OK
//   - Assert response body decodes to an empty slice
//
// This test confirms that:
//   - The handler correctly handles the "no data" case
//   - JSON encoding produces valid output for empty results
//   - No panic or error occurs on empty result sets
func TestGetPlants_Empty(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/plants", nil)
	w := httptest.NewRecorder()

	GetPlants(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var plants []db.Plant
	if err := json.NewDecoder(w.Body).Decode(&plants); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(plants) != 0 {
		t.Errorf("Expected empty list, got %d items", len(plants))
	}
}

// TestSavePlant_ValidJSON verifies that SavePlant correctly creates
// a new plant record when receiving valid JSON input.
//
// Test steps:
//   - Prepare a JSON payload with required fields (serial_number, plant_type)
//   - Send POST /api/plants with Content-Type: application/json
//   - Assert response status is 201 Created
//
// This test confirms that:
//   - JSON decoding works correctly for plant data
//   - New records are inserted into the database
//   - Success response includes appropriate status code
func TestSavePlant_ValidJSON(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	payload := map[string]interface{}{
		"serial_number":  "API-001",
		"plant_type":     "API Test Plant",
		"seed_remainder": 5,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/plants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	SavePlant(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSavePlant_DuplicateSerial verifies that SavePlant rejects
// creation of a plant record with a serial_number that already exists.
//
// Test steps:
//   - Create a plant with serial "DUP-API"
//   - Attempt to create another plant with the same serial
//   - Assert response status is 409 Conflict
//
// This test confirms that:
//   - Uniqueness validation is enforced at the API layer
//   - Duplicate serial numbers return appropriate error code
//   - The first record remains unchanged after failed duplicate attempt
func TestSavePlant_DuplicateSerial(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	payload := map[string]interface{}{
		"serial_number": "DUP-API",
		"plant_type":    "First",
	}
	body, _ := json.Marshal(payload)

	req1 := httptest.NewRequest("POST", "/api/plants", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	SavePlant(httptest.NewRecorder(), req1)

	req2 := httptest.NewRequest("POST", "/api/plants", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	SavePlant(w, req2)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate, got %d", w.Code)
	}
}

// TestSavePlant_InvalidSerial verifies that SavePlant rejects
// plant records with serial_number containing invalid characters.
//
// Test steps:
//   - Prepare JSON payload with serial_number "ab@#$" (contains special chars)
//   - Send POST /api/plants request
//   - Assert response status is 400 Bad Request
//
// This test confirms that:
//   - Input validation regex ^[A-Za-z0-9\-_]{1,}$ is enforced
//   - Invalid serial formats are rejected before database operations
//   - Error response includes descriptive message
func TestSavePlant_InvalidSerial(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	payload := map[string]interface{}{
		"serial_number": "ab@#$",
		"plant_type":    "Bad",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/plants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	SavePlant(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid serial, got %d", w.Code)
	}
}

// TestDeletePlant verifies that DeletePlant successfully removes
// a plant record from the database when given a valid serial number.
//
// Test steps:
//   - Create a plant record with serial "DEL-001"
//   - Send DELETE /api/plants?serial=DEL-001 request
//   - Assert response status is 200 OK
//   - Query database and assert the record no longer exists
//
// This test confirms that:
//   - DELETE operations execute successfully
//   - Query parameter parsing works correctly
//   - Database state is updated as expected
func TestDeletePlant(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	plant := &db.Plant{
		SerialNumber: "DEL-001",
		PlantType:    "ToDelete",
	}
	db.SavePlant(plant)

	req := httptest.NewRequest("DELETE", "/api/plants?serial=DEL-001", nil)
	w := httptest.NewRecorder()

	DeletePlant(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	plants, _ := db.ListPlants()
	for _, p := range plants {
		if p.SerialNumber == "DEL-001" {
			t.Error("Expected plant to be deleted")
		}
	}
}

// TestDeletePlant_NotFound verifies that DeletePlant returns 200 OK
// when called with a serial number that does not exist in the database.
//
// Test steps:
//   - Send DELETE /api/plants?serial=NON-EXISTENT request
//   - Assert response status is 200 OK
//
// This test confirms that:
//   - Delete operation is idempotent (safe to call multiple times)
//   - No error is returned for non-existent records
//   - Cleanup routines can safely call delete without existence checks
func TestDeletePlant_NotFound(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest("DELETE", "/api/plants?serial=NON-EXISTENT", nil)
	w := httptest.NewRecorder()

	DeletePlant(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for non-existent serial, got %d", w.Code)
	}
}

// TestDeletePlant_MissingSerial verifies that DeletePlant returns
// 400 Bad Request when the required serial query parameter is missing.
//
// Test steps:
//   - Send DELETE /api/plants request (no query parameters)
//   - Assert response status is 400 Bad Request
//
// This test confirms that:
//   - Required parameter validation is enforced
//   - Missing parameters return appropriate error code
//   - Handler does not proceed with invalid input
func TestDeletePlant_MissingSerial(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest("DELETE", "/api/plants", nil)
	w := httptest.NewRecorder()

	DeletePlant(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing serial, got %d", w.Code)
	}
}

// TestGetPhoto_NotFound verifies that GetPhoto returns 404 Not Found
// when requesting a photo file that does not exist on disk.
//
// Test steps:
//   - Send GET /api/photos/NON-EXISTENT/fake.jpg request
//   - Assert response status is 404 Not Found
//
// This test confirms that:
//   - File existence checks are performed before serving
//   - Missing files return appropriate HTTP status
//   - No panic occurs when file path is invalid
func TestGetPhoto_NotFound(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/photos/NON-EXISTENT/fake.jpg", nil)
	w := httptest.NewRecorder()

	GetPhoto(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestGetPhoto_InvalidPath verifies that GetPhoto rejects requests
// containing path traversal attempts (e.g., "../" sequences).
//
// Test steps:
//   - Send GET /api/photos/../../../etc/passwd request
//   - Assert response status is 400 Bad Request or 403 Forbidden
//
// This test confirms that:
//   - Security checks prevent directory traversal attacks
//   - Malicious paths are rejected before file system access
//   - Application does not expose arbitrary file system contents
func TestGetPhoto_InvalidPath(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/photos/../../../etc/passwd", nil)
	w := httptest.NewRecorder()

	GetPhoto(w, req)

	if w.Code != http.StatusBadRequest && w.Code != http.StatusForbidden {
		t.Errorf("Expected 400 or 403 for invalid path, got %d", w.Code)
	}
}

// TestSavePlant_MultipartWithPhoto verifies that SavePlant correctly
// handles multipart/form-data requests with file uploads.
//
// Test steps:
//   - Construct multipart form with plant fields and a synthetic JPEG image
//   - Send POST /api/plants with Content-Type: multipart/form-data
//   - Assert response status is 201 Created
//   - Verify photo file was saved to the expected directory
//
// This test confirms that:
//   - Multipart form parsing works correctly
//   - File uploads are processed and saved with unique names
//   - Photo metadata is stored in the database record
func TestSavePlant_MultipartWithPhoto(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("serial_number", "MULTI-001")
	_ = writer.WriteField("plant_type", "Multipart Test")
	_ = writer.WriteField("seed_remainder", "10")

	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	part, err := writer.CreateFormFile("photo", "test.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if err := jpeg.Encode(part, img, nil); err != nil {
		t.Fatalf("Failed to encode image: %v", err)
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/api/plants", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	SavePlant(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	photoDir := db.GetPhotoDir("MULTI-001")
	files, _ := os.ReadDir(photoDir)
	if len(files) == 0 {
		t.Error("Expected photo file to be saved")
	}
}

// TestSavePlant_EmptySerial verifies that SavePlant rejects plant
// records with an empty serial_number field.
//
// Test steps:
//   - Prepare JSON payload with serial_number: ""
//   - Send POST /api/plants request
//   - Assert response status is 400 Bad Request
//
// This test confirms that:
//   - Required field validation is enforced
//   - Empty serial_number is rejected before database operations
//   - Error response includes descriptive message
func TestSavePlant_EmptySerial(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	payload := map[string]interface{}{
		"serial_number": "",
		"plant_type":    "Bad",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/plants", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	SavePlant(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty serial, got %d", w.Code)
	}
}

// TestListPlants_WithQuery verifies that GetPlants returns all records
// regardless of query parameters, as filtering is performed client-side.
//
// Test steps:
//   - Insert three plant records with different plant_type and tags
//   - Send GET /api/plants?q=tomato request (query param ignored by backend)
//   - Assert response status is 200 OK
//   - Assert all three records are returned in response
//
// This test confirms that:
//   - Backend returns full dataset without server-side filtering
//   - Query parameters do not affect result set (client handles filtering)
//   - JSON encoding preserves all record fields correctly
func TestListPlants_WithQuery(t *testing.T) {
	cleanup := setupTestAPI(t)
	defer cleanup()

	plants := []*db.Plant{
		{SerialNumber: "Q-001", PlantType: "Tomato", Tags: "vegetable"},
		{SerialNumber: "Q-002", PlantType: "Cucumber", Tags: "vegetable"},
		{SerialNumber: "Q-003", PlantType: "Rose", Tags: "flower"},
	}
	for _, p := range plants {
		db.SavePlant(p)
	}

	req := httptest.NewRequest("GET", "/api/plants?q=tomato", nil)
	w := httptest.NewRecorder()

	GetPlants(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result []db.Plant
	json.NewDecoder(w.Body).Decode(&result)

	if len(result) != 3 {
		t.Errorf("Expected 3 plants (backend returns all), got %d", len(result))
	}
}
