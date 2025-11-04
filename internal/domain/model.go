package domain

import "time"

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
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type UpdateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}
