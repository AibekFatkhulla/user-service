package domain

import (
	"errors"
	"time"
)

// User errors
var (
	ErrUserNotFound                = errors.New("user not found")
	ErrInsufficientCoinsBalance    = errors.New("insufficient coins balance")
	ErrSubscriptionAlreadyActive   = errors.New("subscription already active")
	ErrNoActiveSubscription        = errors.New("user does not have an active subscription")
	ErrInvalidCoinsAmount          = errors.New("coins must be greater than 0")
	ErrInvalidEmailFormat          = errors.New("invalid email format")
	ErrEmailAlreadyExists          = errors.New("user with this email already exists")
	ErrInvalidStatus               = errors.New("invalid status")
	ErrInvalidSubscriptionDuration = errors.New("subscription duration must be greater than 0")
	ErrEmailRequired               = errors.New("email is required")
	ErrNameRequired                = errors.New("name is required")
	ErrUserIDRequired              = errors.New("user ID is required")
	ErrEmailTooLong                = errors.New("email is too long")
	ErrNameTooLong                 = errors.New("name is too long")
	ErrInvalidUUID                 = errors.New("invalid user ID format")
	ErrCoinsAmountTooLarge         = errors.New("coins amount is too large")
	ErrListLimitTooLarge           = errors.New("list limit is too large")
	ErrListOffsetTooLarge          = errors.New("list offset is too large")
	ErrSubscriptionDurationTooLong = errors.New("subscription duration is too long")
)

// User status constants
const (
	StatusActive    = "active"
	StatusInactive  = "inactive"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"
)

// Validation constants
const (
	MaxEmailLength     = 255
	MaxNameLength      = 100
	MaxCoinsAmount     = 1_000_000_000 // 1 billion
	MaxListLimit       = 100
	MaxListOffset      = 10_000_000      // 10 million
	MaxRequestBodySize = 1 * 1024 * 1024 // 1 MB
	MaxSubscriptionDurationHours = 87600 // 10 years (365 * 24 * 10)
)

// ValidStatuses returns list of valid user statuses
func ValidStatuses() []string {
	return []string{StatusActive, StatusInactive, StatusSuspended, StatusDeleted}
}

type User struct {
	ID                  string     `json:"id"`
	Email               string     `json:"email"`
	Name                string     `json:"name"`
	CoinsBalance        int64      `json:"coins_balance"`
	TotalCoinsPurchased int64      `json:"total_coins_purchased"`
	IsTrial             bool       `json:"is_trial"`
	TrialEndsAt         *time.Time `json:"trial_ends_at"`
	HasSubscription     bool       `json:"has_subscription"`
	SubscriptionEndsAt  *time.Time `json:"subscription_ends_at"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type CreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type UpdateUserRequest struct {
	Email  string  `json:"email"`
	Name   string  `json:"name"`
	Status *string `json:"status"` // optional
}

// UpdateUserFields represents fields to update in repository
// nil pointer means "don't update this field"
type UpdateUserFields struct {
	Email  *string
	Name   *string
	Status *string
}
