// Package db provides data model definitions for the Svenskt Vin database.
package db

import "time"

// User represents a registered user account.
type User struct {
	ID             int64      `json:"id"`
	Email          string     `json:"email"`
	PasswordHash   *string    `json:"-"` // Never expose in JSON
	Name           string     `json:"name"`
	IsAdmin        bool       `json:"is_admin"`
	Active         bool       `json:"active"`
	LastLogin      *time.Time `json:"last_login,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Session represents an active user session.
type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Vineyard represents a vineyard property.
type Vineyard struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	County          string    `json:"county"`
	Municipality    string    `json:"municipality"`
	Lat             float64   `json:"lat"`
	Lon             float64   `json:"lon"`
	EstablishedYear *int      `json:"established_year"`
	TotalAreaHA     *float64  `json:"total_area_ha"`
	Organic         bool      `json:"organic"`
	Biodynamic      bool      `json:"biodynamic"`
	LegalID         *string   `json:"legal_id"`
	LegalIDType     *string   `json:"legal_id_type"`
	LegalName       *string   `json:"legal_name"`
	DeletedAt       *time.Time `json:"deleted_at"`
}

// VineyardMember represents a user's membership in a vineyard.
type VineyardMember struct {
	VineyardID int64  `json:"vineyard_id"`
	UserID     int64  `json:"user_id"`
	Role       string `json:"role"` // "owner", "editor"
}

// Block represents a vineyard block (parcel).
type Block struct {
	ID             int64     `json:"id"`
	VineyardID     int64     `json:"vineyard_id"`
	VarietyID      int64     `json:"variety_id"`
	BlockName      string    `json:"block_name"`
	AreaHA         float64   `json:"area_ha"`
	VineCount      *int      `json:"vine_count"`
	PlantingYear   *int      `json:"planting_year"`
	TrainingSystem *string   `json:"training_system"`
	Aspect         *string   `json:"aspect"`
	SlopeDegrees   *float64  `json:"slope_degrees"`
	ElevationM     *int      `json:"elevation_m"`
	IsActive       bool      `json:"is_active"`
	DeletedAt      *time.Time `json:"deleted_at"`
}

// HarvestRecord represents a harvest record for a block.
type HarvestRecord struct {
	ID               int64     `json:"id"`
	BlockID          int64     `json:"block_id"`
	HarvestYear      *int      `json:"harvest_year"`
	HarvestDate      *string   `json:"harvest_date"`
	YieldKG          float64   `json:"yield_kg"`
	Brix             *float64  `json:"brix"`
	AcidGL           *float64  `json:"acid_g_l"`
	VineHealthRating *int      `json:"vine_health_rating"`
	Notes            *string   `json:"notes"`
	StillWineL       *float64  `json:"still_wine_l"`
	SparklingL       *float64  `json:"sparkling_l"`
	JuiceL           *float64  `json:"juice_l"`
	SoldKG           *float64  `json:"sold_kg"`
	DiscardedKG      *float64  `json:"discarded_kg"`
	DeletedAt        *time.Time `json:"deleted_at"`
}

// Variety represents a grape variety.
type Variety struct {
	ID                   int64     `json:"id"`
	Name                 string    `json:"name"`
	Piwi                 bool      `json:"piwi"`
	Color                string    `json:"color"`
	Status               string    `json:"status"` // "approved", "review_needed"
	SubmittedByVineyardID *int64   `json:"submitted_by_vineyard_id"`
}

// MagicLinkToken represents a one-time magic link token.
type MagicLinkToken struct {
	UserID    int64     `json:"user_id"`
	TokenHash []byte    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}

// PendingInvite represents a pending vineyard membership invite.
type PendingInvite struct {
	ID          int64     `json:"id"`
	Email       string    `json:"email"`
	VineyardID  int64     `json:"vineyard_id"`
	Token       string    `json:"token"`
	CompanyName string    `json:"company_name"`
	OwnerName   string    `json:"owner_name"`
	Role        string    `json:"role"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// BlockLock represents an active block lock for harvest creation.
type BlockLock struct {
	ID        int64     `json:"id"`
	BlockID   int64     `json:"block_id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Harvest represents a harvest record (alias for HarvestRecord for handler use).
type Harvest struct {
	ID               int64      `json:"id"`
	BlockID          int64      `json:"block_id"`
	HarvestYear      *int       `json:"harvest_year"`
	HarvestDate      *time.Time `json:"harvest_date"`
	YieldKG          float64    `json:"yield_kg"`
	Brix             *float64   `json:"brix"`
	AcidgL           *float64   `json:"acid_g_l"`
	VineHealthRating *int       `json:"vine_health_rating"`
	Notes            *string    `json:"notes"`
	StillWineL       *float64   `json:"still_wine_l"`
	SparklingL       *float64   `json:"sparkling_l"`
	JuiceL           *float64   `json:"juice_l"`
	SoldKG           *float64   `json:"sold_kg"`
	DiscardedKG      *float64   `json:"discarded_kg"`
}

// HarvestCreateInput holds input data for creating a harvest record.
type HarvestCreateInput struct {
	BlockID          int64      `json:"block_id"`
	VineyardID       int64      `json:"vineyard_id"`
	UserID           int64      `json:"user_id"`
	HarvestDate      *time.Time `json:"harvest_date"`
	HarvestYear      int        `json:"harvest_year"`
	YieldKG          float64    `json:"yield_kg"`
	Brix             *float64   `json:"brix"`
	AcidgL           *float64   `json:"acid_g_l"`
	VineHealthRating *int       `json:"vine_health_rating"`
	Notes            *string    `json:"notes"`
	StillWineL       *float64   `json:"still_wine_l"`
	SparklingL       *float64   `json:"sparkling_l"`
	JuiceL           *float64   `json:"juice_l"`
	SoldKG           *float64   `json:"sold_kg"`
	DiscardedKG      *float64   `json:"discarded_kg"`
}

// HarvestUpdateInput holds optional fields for updating a harvest record.
type HarvestUpdateInput struct {
	HarvestDate      *time.Time `json:"harvest_date"`
	HarvestYear      *int       `json:"harvest_year"`
	YieldKG          *float64   `json:"yield_kg"`
	Brix             *float64   `json:"brix"`
	AcidgL           *float64   `json:"acid_g_l"`
	VineHealthRating *int       `json:"vine_health_rating"`
	Notes            *string    `json:"notes"`
	StillWineL       *float64   `json:"still_wine_l"`
	SparklingL       *float64   `json:"sparkling_l"`
	JuiceL           *float64   `json:"juice_l"`
	SoldKG           *float64   `json:"sold_kg"`
	DiscardedKG      *float64   `json:"discarded_kg"`
}

// BlockLockInput holds input data for creating a block lock.
type BlockLockInput struct {
	BlockID int64 `json:"block_id"`
	UserID  int64 `json:"user_id"`
}
