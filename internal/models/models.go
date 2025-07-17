package models

import (
	"time"
)

type User struct {
	ID           int       `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Currency     string    `json:"currency" db:"currency"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Category struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Item struct {
	ID            int       `json:"id" db:"id"`
	UserID        int       `json:"user_id" db:"user_id"`
	CategoryID    int       `json:"category_id" db:"category_id"`
	Name          string    `json:"name" db:"name"`
	Note          string    `json:"note" db:"note"`
	WeightGrams   int       `json:"weight_grams" db:"weight_grams"`
	Price         float64   `json:"price" db:"price"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	Category      *Category `json:"category,omitempty"`
}

type Pack struct {
	ID        string    `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	IsPublic  bool      `json:"is_public" db:"is_public"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Items     []PackItem `json:"items,omitempty"`
}

type PackItem struct {
	ID        int  `json:"id" db:"id"`
	PackID    string `json:"pack_id" db:"pack_id"`
	ItemID    int  `json:"item_id" db:"item_id"`
	IsWorn    bool `json:"is_worn" db:"is_worn"`
	Count     int  `json:"count" db:"count"`
	WornCount int  `json:"worn_count" db:"worn_count"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Item      *Item `json:"item,omitempty"`
}

type Session struct {
	ID        string    `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type CSRFToken struct {
	Token     string    `json:"token" db:"token"`
	UserID    int       `json:"user_id" db:"user_id"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

