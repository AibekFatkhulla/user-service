package domain

import (
	"errors"
	"time"
)

// User errors
var (
	ErrUserNotFound = errors.New("user not found")
)

// User status constants
const (
	StatusActive    = "active"
	StatusInactive  = "inactive"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"
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
