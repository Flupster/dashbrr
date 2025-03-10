// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"os"
	"testing"

	"github.com/autobrr/dashbrr/internal/models"
	"github.com/autobrr/dashbrr/internal/types"
)

// setupTestDB sets up a SQLite test database
func setupTestDB(t *testing.T) (*DB, func()) {
	var db *DB
	var err error

	// Set SQLite test environment
	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"
	os.Setenv("DASHBRR__DB_TYPE", "sqlite")
	os.Setenv("DASHBRR__DB_PATH", dbPath)

	cleanup := func() {
		if db != nil {
			db.Close()
		}
		os.Remove(dbPath)
		os.Unsetenv("DASHBRR__DB_TYPE")
		os.Unsetenv("DASHBRR__DB_PATH")
	}

	config := NewConfig()
	db, err = InitDBWithConfig(config)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	return db, cleanup
}

func TestDatabaseInitialization(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test database connection
	err := db.Ping()
	if err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}

	// Verify schema initialization
	var tableCount int
	query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table'"

	err = db.QueryRow(query).Scan(&tableCount)
	if err != nil {
		t.Errorf("Failed to count tables: %v", err)
	}

	// We expect at least the users and service_configurations tables
	if tableCount < 2 {
		t.Errorf("Expected at least 2 tables, got %d", tableCount)
	}
}

func TestUserOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test user creation
	user := &types.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
	}

	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("Expected user ID to be set after creation")
	}

	// Test user retrieval by username
	retrieved, err := db.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("Failed to get user by username: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to find user, got nil")
	}

	if retrieved.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
	}

	// Test user retrieval by email
	retrieved, err = db.GetUserByEmail("test@example.com")
	if err != nil {
		t.Fatalf("Failed to get user by email: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to find user, got nil")
	}

	if retrieved.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, retrieved.Username)
	}

	// Test HasUsers
	hasUsers, err := db.HasUsers()
	if err != nil {
		t.Fatalf("Failed to check if has users: %v", err)
	}

	if !hasUsers {
		t.Error("Expected HasUsers to return true after creating a user")
	}
}

func TestServiceOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test service creation
	service := &models.ServiceConfiguration{
		InstanceID:  "test-service-1",
		DisplayName: "Test Service",
		URL:         "http://localhost:8080",
		APIKey:      "test-api-key",
	}

	err := db.CreateService(service)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if service.ID == 0 {
		t.Error("Expected service ID to be set after creation")
	}

	// Test service retrieval by instance ID
	retrieved, err := db.GetServiceByInstanceID("test-service-1")
	if err != nil {
		t.Fatalf("Failed to get service by instance ID: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to find service, got nil")
	}

	if retrieved.DisplayName != service.DisplayName {
		t.Errorf("Expected display name %s, got %s", service.DisplayName, retrieved.DisplayName)
	}

	// Test service update
	service.DisplayName = "Updated Test Service"
	err = db.UpdateService(service)
	if err != nil {
		t.Fatalf("Failed to update service: %v", err)
	}

	retrieved, err = db.GetServiceByInstanceID("test-service-1")
	if err != nil {
		t.Fatalf("Failed to get updated service: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to find updated service, got nil")
	}

	if retrieved.DisplayName != "Updated Test Service" {
		t.Errorf("Expected updated display name %s, got %s", "Updated Test Service", retrieved.DisplayName)
	}

	// Test GetAllServices
	services, err := db.GetAllServices()
	if err != nil {
		t.Fatalf("Failed to get all services: %v", err)
	}

	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	// Test service deletion
	err = db.DeleteService("test-service-1")
	if err != nil {
		t.Fatalf("Failed to delete service: %v", err)
	}

	retrieved, err = db.GetServiceByInstanceID("test-service-1")
	if err != nil {
		t.Fatalf("Failed to check deleted service: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected service to be deleted")
	}
}

func TestErrorHandling(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test duplicate user creation
	user1 := &types.User{
		Username:     "duplicate",
		Email:        "duplicate@example.com",
		PasswordHash: "hashedpassword",
	}

	err := db.CreateUser(user1)
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	user2 := &types.User{
		Username:     "duplicate",
		Email:        "duplicate@example.com",
		PasswordHash: "hashedpassword",
	}

	err = db.CreateUser(user2)
	if err == nil {
		t.Error("Expected error when creating duplicate user, got nil")
	}

	// Test duplicate service creation
	service1 := &models.ServiceConfiguration{
		InstanceID:  "duplicate-service",
		DisplayName: "Duplicate Service",
		URL:         "http://localhost:8080",
		APIKey:      "test-api-key",
	}

	err = db.CreateService(service1)
	if err != nil {
		t.Fatalf("Failed to create first service: %v", err)
	}

	service2 := &models.ServiceConfiguration{
		InstanceID:  "duplicate-service",
		DisplayName: "Duplicate Service",
		URL:         "http://localhost:8080",
		APIKey:      "test-api-key",
	}

	err = db.CreateService(service2)
	if err == nil {
		t.Error("Expected error when creating duplicate service, got nil")
	}
}
