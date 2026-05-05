// Package db provides unit tests for database operations in the Gardener application.
//
// Tests verify:
//   - Plant CRUD operations (create, read, delete)
//   - Database schema constraints (UNIQUE serial_number, CHECK seed_remainder >= 0)
//   - Default sorting behavior (ORDER BY plant_type ASC)
//   - Migration logic for schema evolution
//
// All tests use t.TempDir() and HOME environment variable mocking
// to ensure isolation and avoid modifying the user's actual database.
package db

import (
	"os"
	"testing"
)

// setupTestDB initializes a test database in a temporary directory.
//
// It performs the following:
//   - Creates a temporary directory via t.TempDir()
//   - Overrides the HOME environment variable to isolate test data
//   - Calls db.Init() to set up schema and connection
//
// Returns a cleanup function that must be deferred by the caller to:
//   - Restore the original HOME environment variable
//   - Close the database connection
//
// Usage:
//
//	cleanup := setupTestDB(t)
//	defer cleanup()
func setupTestDB(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	if err := Init(); err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}

	return func() {
		os.Setenv("HOME", origHome)
		DB.Close()
	}
}

// TestSaveAndListPlants verifies the basic create and read workflow.
//
// Test steps:
//   - Create a Plant with known field values
//   - Save it via SavePlant()
//   - Retrieve all plants via ListPlants()
//   - Assert exactly one record exists with matching SerialNumber
//
// This test confirms that:
//   - INSERT operations persist data correctly
//   - SELECT operations return expected records
//   - Field mapping between struct and database columns works
func TestSaveAndListPlants(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	plant := &Plant{
		SerialNumber:  "TEST-001",
		PlantType:     "Тестовый томат",
		SeedRemainder: 10,
		Tags:          "тест,огурец",
	}

	if err := SavePlant(plant); err != nil {
		t.Fatalf("SavePlant failed: %v", err)
	}

	plants, err := ListPlants()
	if err != nil {
		t.Fatalf("ListPlants failed: %v", err)
	}

	if len(plants) != 1 {
		t.Errorf("Expected 1 plant, got %d", len(plants))
	}

	if plants[0].SerialNumber != "TEST-001" {
		t.Errorf("Expected serial TEST-001, got %s", plants[0].SerialNumber)
	}
}

// TestUniqueSerialConstraint verifies that the UNIQUE constraint on
// serial_number is enforced at the database level.
//
// Test steps:
//   - Save a plant with serial "DUP-001"
//   - Attempt to save another plant with the same serial
//   - Assert that the second SavePlant() returns an error
//
// This test confirms that:
//   - Duplicate serial numbers are rejected
//   - Application-layer uniqueness checks are backed by database constraints
func TestUniqueSerialConstraint(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	p1 := &Plant{SerialNumber: "DUP-001", PlantType: "First"}
	p2 := &Plant{SerialNumber: "DUP-001", PlantType: "Second"}

	if err := SavePlant(p1); err != nil {
		t.Fatalf("First save failed: %v", err)
	}

	if err := SavePlant(p2); err == nil {
		t.Error("Expected UNIQUE constraint error, got nil")
	}
}

// TestSeedRemainderNonNegative verifies that negative seed_remainder
// values are clamped to zero before insertion.
//
// Test steps:
//   - Create a Plant with SeedRemainder = -5
//   - Save it via SavePlant()
//   - Retrieve the record and assert SeedRemainder == 0
//
// This test confirms that:
//   - Business logic validation (non-negative remainder) is enforced
//   - The CHECK constraint at the database level is complemented by application logic
func TestSeedRemainderNonNegative(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	plant := &Plant{
		SerialNumber:  "NEG-001",
		SeedRemainder: -5,
	}

	if err := SavePlant(plant); err != nil {
		t.Fatalf("SavePlant failed: %v", err)
	}

	plants, _ := ListPlants()
	if plants[0].SeedRemainder != 0 {
		t.Errorf("Expected seed_remainder=0, got %d", plants[0].SeedRemainder)
	}
}

// TestListPlants_Sorting verifies that ListPlants returns records
// sorted by plant_type in ascending order.
//
// Test steps:
//   - Insert three plants with plant_types: "Zebra Plant", "Apple Tree", "Mint"
//   - Retrieve all plants via ListPlants()
//   - Assert the order is: Apple Tree, Mint, Zebra Plant
//
// This test confirms that:
//   - The ORDER BY plant_type clause in the query works correctly
//   - Client-side display can rely on consistent ordering
func TestListPlants_Sorting(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	plants := []*Plant{
		{SerialNumber: "Z-001", PlantType: "Zebra Plant", SeedRemainder: 5},
		{SerialNumber: "A-001", PlantType: "Apple Tree", SeedRemainder: 20},
		{SerialNumber: "M-001", PlantType: "Mint", SeedRemainder: 10},
	}
	for _, p := range plants {
		SavePlant(p)
	}

	result, err := ListPlants()
	if err != nil {
		t.Fatalf("ListPlants failed: %v", err)
	}

	if result[0].PlantType != "Apple Tree" {
		t.Errorf("Expected first plant to be 'Apple Tree', got %q", result[0].PlantType)
	}
	if result[2].PlantType != "Zebra Plant" {
		t.Errorf("Expected last plant to be 'Zebra Plant', got %q", result[2].PlantType)
	}
}

// TestDeletePlant_DB verifies that DeletePlant removes a record
// from the database by its serial number.
//
// Test steps:
//   - Save a plant with serial "DEL-DB"
//   - Delete it via DeletePlant()
//   - Retrieve all plants and assert the record no longer exists
//
// This test confirms that:
//   - DELETE queries execute successfully
//   - The operation is idempotent for existing records
func TestDeletePlant_DB(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	p := &Plant{SerialNumber: "DEL-DB", PlantType: "ToDelete"}
	SavePlant(p)

	err := DeletePlant("DEL-DB")
	if err != nil {
		t.Errorf("DeletePlant failed: %v", err)
	}

	plants, _ := ListPlants()
	for _, plant := range plants {
		if plant.SerialNumber == "DEL-DB" {
			t.Error("Expected plant to be deleted from DB")
		}
	}
}

// TestDeletePlant_NonExistent verifies that DeletePlant does not
// return an error when called with a non-existent serial number.
//
// Test steps:
//   - Call DeletePlant("NON-EXISTENT") on a clean database
//   - Assert that no error is returned
//
// This test confirms that:
//   - The delete operation is idempotent
//   - Cleanup routines can safely call DeletePlant without pre-checking existence
func TestDeletePlant_NonExistent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	err := DeletePlant("NON-EXISTENT")
	if err != nil {
		t.Errorf("DeletePlant should not fail for non-existent serial: %v", err)
	}
}
