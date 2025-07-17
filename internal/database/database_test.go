package database

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"carryless/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal("Failed to open test database:", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatal("Failed to run migrations:", err)
	}

	return db
}

func TestUserCreationAndAuthentication(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", user.Username)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", user.Email)
	}

	authUser, err := AuthenticateUser(db, "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to authenticate user:", err)
	}

	if authUser.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, authUser.ID)
	}

	_, err = AuthenticateUser(db, "test@example.com", "wrongpassword")
	if err == nil {
		t.Error("Expected authentication to fail with wrong password")
	}
}

func TestSessionManagement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	session, err := CreateSession(db, user.ID)
	if err != nil {
		t.Fatal("Failed to create session:", err)
	}

	if len(session.ID) == 0 {
		t.Error("Session ID should not be empty")
	}

	validatedUser, err := ValidateSession(db, session.ID)
	if err != nil {
		t.Fatal("Failed to validate session:", err)
	}

	if validatedUser.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, validatedUser.ID)
	}

	err = DeleteSession(db, session.ID)
	if err != nil {
		t.Fatal("Failed to delete session:", err)
	}

	_, err = ValidateSession(db, session.ID)
	if err == nil {
		t.Error("Expected session validation to fail after deletion")
	}
}

func TestCategoryOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	category, err := CreateCategory(db, user.ID, "Sleeping")
	if err != nil {
		t.Fatal("Failed to create category:", err)
	}

	if category.Name != "Sleeping" {
		t.Errorf("Expected category name 'Sleeping', got %s", category.Name)
	}

	categories, err := GetCategories(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get categories:", err)
	}

	if len(categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(categories))
	}

	err = UpdateCategory(db, user.ID, category.ID, "Sleep System")
	if err != nil {
		t.Fatal("Failed to update category:", err)
	}

	updatedCategory, err := GetCategory(db, user.ID, category.ID)
	if err != nil {
		t.Fatal("Failed to get updated category:", err)
	}

	if updatedCategory.Name != "Sleep System" {
		t.Errorf("Expected category name 'Sleep System', got %s", updatedCategory.Name)
	}

	err = DeleteCategory(db, user.ID, category.ID)
	if err != nil {
		t.Fatal("Failed to delete category:", err)
	}

	_, err = GetCategory(db, user.ID, category.ID)
	if err == nil {
		t.Error("Expected category retrieval to fail after deletion")
	}
}

func TestItemOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	category, err := CreateCategory(db, user.ID, "Sleeping")
	if err != nil {
		t.Fatal("Failed to create category:", err)
	}

	item := models.Item{
		CategoryID:   category.ID,
		Name:         "Sleeping Bag",
		Note:         "Down sleeping bag",
		WeightGrams:  800,
		Price:        299.99,
	}

	createdItem, err := CreateItem(db, user.ID, item)
	if err != nil {
		t.Fatal("Failed to create item:", err)
	}

	if createdItem.Name != "Sleeping Bag" {
		t.Errorf("Expected item name 'Sleeping Bag', got %s", createdItem.Name)
	}

	items, err := GetItems(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get items:", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}

	updatedItem := models.Item{
		CategoryID:   category.ID,
		Name:         "Down Sleeping Bag",
		Note:         "Lightweight down sleeping bag",
		WeightGrams:  750,
		Price:        299.99,
	}

	err = UpdateItem(db, user.ID, createdItem.ID, updatedItem)
	if err != nil {
		t.Fatal("Failed to update item:", err)
	}

	retrievedItem, err := GetItem(db, user.ID, createdItem.ID)
	if err != nil {
		t.Fatal("Failed to get updated item:", err)
	}

	if retrievedItem.Name != "Down Sleeping Bag" {
		t.Errorf("Expected item name 'Down Sleeping Bag', got %s", retrievedItem.Name)
	}

	if retrievedItem.WeightGrams != 750 {
		t.Errorf("Expected weight 750g, got %dg", retrievedItem.WeightGrams)
	}

	err = DeleteItem(db, user.ID, createdItem.ID)
	if err != nil {
		t.Fatal("Failed to delete item:", err)
	}

	_, err = GetItem(db, user.ID, createdItem.ID)
	if err == nil {
		t.Error("Expected item retrieval to fail after deletion")
	}
}

func TestPackOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	pack, err := CreatePack(db, user.ID, "Weekend Trip")
	if err != nil {
		t.Fatal("Failed to create pack:", err)
	}

	if pack.Name != "Weekend Trip" {
		t.Errorf("Expected pack name 'Weekend Trip', got %s", pack.Name)
	}

	if len(pack.ID) == 0 {
		t.Error("Pack ID should not be empty")
	}

	packs, err := GetPacks(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get packs:", err)
	}

	if len(packs) != 1 {
		t.Errorf("Expected 1 pack, got %d", len(packs))
	}

	err = UpdatePack(db, user.ID, pack.ID, "Extended Weekend Trip", true)
	if err != nil {
		t.Fatal("Failed to update pack:", err)
	}

	updatedPack, err := GetPack(db, pack.ID)
	if err != nil {
		t.Fatal("Failed to get updated pack:", err)
	}

	if updatedPack.Name != "Extended Weekend Trip" {
		t.Errorf("Expected pack name 'Extended Weekend Trip', got %s", updatedPack.Name)
	}

	if !updatedPack.IsPublic {
		t.Error("Expected pack to be public")
	}

	err = DeletePack(db, user.ID, pack.ID)
	if err != nil {
		t.Fatal("Failed to delete pack:", err)
	}

	_, err = GetPack(db, pack.ID)
	if err == nil {
		t.Error("Expected pack retrieval to fail after deletion")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}