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
	IsAdmin      bool      `json:"is_admin" db:"is_admin"`
	IsActivated  bool      `json:"is_activated" db:"is_activated"`
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
	ID             int        `json:"id" db:"id"`
	UserID         int        `json:"user_id" db:"user_id"`
	CategoryID     int        `json:"category_id" db:"category_id"`
	Name           string     `json:"name" db:"name"`
	Note           string     `json:"note" db:"note"`
	WeightGrams    int        `json:"weight_grams" db:"weight_grams"`
	WeightToVerify bool       `json:"weight_to_verify" db:"weight_to_verify"`
	Price          float64    `json:"price" db:"price"`
	Brand          *string    `json:"brand,omitempty" db:"brand"`
	Model          *string    `json:"model,omitempty" db:"model"`
	PurchaseDate   *time.Time `json:"purchase_date,omitempty" db:"purchase_date"`
	Capacity       *float64   `json:"capacity,omitempty" db:"capacity"`
	CapacityUnit   *string    `json:"capacity_unit,omitempty" db:"capacity_unit"`
	Link           *string    `json:"link,omitempty" db:"link"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	Category       *Category  `json:"category,omitempty"`
	LinkedItems    []ItemLink `json:"linked_items,omitempty"`
	HasLinkedItems bool       `json:"has_linked_items"`
}

type Pack struct {
	ID        string    `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Note      string    `json:"note" db:"note"`
	IsPublic  bool      `json:"is_public" db:"is_public"`
	IsLocked  bool      `json:"is_locked" db:"is_locked"`
	ShortID   string    `json:"short_id,omitempty" db:"short_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Items     []PackItem `json:"items,omitempty"`
	Labels    []PackLabel `json:"labels,omitempty"`
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
	Labels    []ItemLabel `json:"labels,omitempty"`
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

type ActivationToken struct {
	Token     string    `json:"token" db:"token"`
	UserID    int       `json:"user_id" db:"user_id"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type ItemInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PackLabel struct {
	ID        int       `json:"id" db:"id"`
	PackID    string    `json:"pack_id" db:"pack_id"`
	Name      string    `json:"name" db:"name"`
	Color     string    `json:"color" db:"color"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type ItemLabel struct {
	ID           int       `json:"id" db:"id"`
	PackItemID   int       `json:"pack_item_id" db:"pack_item_id"`
	PackLabelID  int       `json:"pack_label_id" db:"pack_label_id"`
	Count        int       `json:"count" db:"count"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	PackLabel    *PackLabel `json:"pack_label,omitempty"`
}

type Trip struct {
	ID             string               `json:"id" db:"id"`
	UserID         int                  `json:"user_id" db:"user_id"`
	Name           string               `json:"name" db:"name"`
	Description    *string              `json:"description,omitempty" db:"description"`
	Location       *string              `json:"location,omitempty" db:"location"`
	StartDate      *time.Time           `json:"start_date,omitempty" db:"start_date"`
	EndDate        *time.Time           `json:"end_date,omitempty" db:"end_date"`
	Notes          *string              `json:"notes,omitempty" db:"notes"`
	GPXData        *string              `json:"gpx_data,omitempty" db:"gpx_data"`
	IsPublic       bool                 `json:"is_public" db:"is_public"`
	IsArchived     bool                 `json:"is_archived" db:"is_archived"`
	ShortID        string               `json:"short_id,omitempty" db:"short_id"`
	CreatedAt      time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at" db:"updated_at"`
	Packs          []Pack               `json:"packs,omitempty"`
	ChecklistItems []TripChecklistItem  `json:"checklist_items,omitempty"`
	TransportSteps []TripTransportStep  `json:"transport_steps,omitempty"`
}

type TripChecklistItem struct {
	ID        int       `json:"id" db:"id"`
	TripID    string    `json:"trip_id" db:"trip_id"`
	Content   string    `json:"content" db:"content"`
	IsChecked bool      `json:"is_checked" db:"is_checked"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type TripTransportStep struct {
	ID                int        `json:"id" db:"id"`
	TripID            string     `json:"trip_id" db:"trip_id"`
	JourneyType       string     `json:"journey_type" db:"journey_type"`
	StepOrder         int        `json:"step_order" db:"step_order"`
	DeparturePlace    string     `json:"departure_place" db:"departure_place"`
	DepartureDatetime *time.Time `json:"departure_datetime,omitempty" db:"departure_datetime"`
	ArrivalPlace      *string    `json:"arrival_place,omitempty" db:"arrival_place"`
	ArrivalDatetime   *time.Time `json:"arrival_datetime,omitempty" db:"arrival_datetime"`
	TransportType     *string    `json:"transport_type,omitempty" db:"transport_type"`
	TransportNumber   *string    `json:"transport_number,omitempty" db:"transport_number"`
	Notes             *string    `json:"notes,omitempty" db:"notes"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// Duration returns the duration of the transport step if both departure and arrival times are set
func (t *TripTransportStep) Duration() *time.Duration {
	if t.DepartureDatetime != nil && t.ArrivalDatetime != nil {
		duration := t.ArrivalDatetime.Sub(*t.DepartureDatetime)
		return &duration
	}
	return nil
}

type ItemLink struct {
	ID           int       `json:"id" db:"id"`
	ParentItemID int       `json:"parent_item_id" db:"parent_item_id"`
	LinkedItemID int       `json:"linked_item_id" db:"linked_item_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	LinkedItem   *Item     `json:"linked_item,omitempty"`
}

