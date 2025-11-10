package domain

import (
	"errors"
	"time"
	"strings"
)

const (
	maxProductNameLength = 200
	maxProductSlugLength = 50
	minProductPrice      = 1
	maxProductPrice      = 1_000_000_000
)

var (
	ErrProductNotFound    = errors.New("product not found")
	ErrProductSlugExists  = errors.New("product slug already exists")
	ErrInvalidProductSlug = errors.New("invalid product slug")
	ErrInvalidProductName = errors.New("invalid product name")
	ErrInvalidPrice       = errors.New("invalid product price")
	ErrProductInactive    = errors.New("product is inactive")
)

type Product struct {
	ID          string    `json:"id"`
	CategoryID  string    `json:"category_id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	PriceCoins  int64     `json:"price_coins"`
	Metadata    string    `json:"metadata,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateProductRequest struct {
	CategoryID  string `json:"category_id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCoins  int64  `json:"price_coins"`
	Metadata    string `json:"metadata,omitempty"`
	IsActive    bool   `json:"is_active"`
}

type UpdateProductRequest struct {
	CategoryID  *string `json:"category_id,omitempty"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	PriceCoins  *int64  `json:"price_coins,omitempty"`
	Metadata    *string `json:"metadata,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

func ValidateProductSlug(slug string) error {
	if slug == "" || len(slug) > maxProductSlugLength {
		return ErrInvalidProductSlug
	}
	if strings.ContainsAny(slug, " ") {
		return ErrInvalidProductSlug
	}
	return nil
}

func ValidateProductName(name string) error {
	if name == "" || len(name) > maxProductNameLength {
		return ErrInvalidProductName
	}
	return nil
}

func ValidateProductPrice(price int64) error {
	if price < minProductPrice || price > maxProductPrice {
		return ErrInvalidPrice
	}
	return nil
}