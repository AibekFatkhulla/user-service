package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	maxCategoryNameLength = 100
	maxCategorySlugLength = 50
)

var (
	ErrCategoryNotFound    = errors.New("product category not found")
	ErrCategorySlugExists  = errors.New("product category slug already exists")
	ErrInvalidCategorySlug = errors.New("invalid product category slug")
	ErrInvalidCategoryName = errors.New("invalid product category name")
)

type ProductCategory struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Position    int       `json:"position"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateCategoryRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	IsActive    bool   `json:"is_active"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Position    *int    `json:"position,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

func ValidateCategorySlug(slug string) error {
	if slug == "" || len(slug) > maxCategorySlugLength {
		return ErrInvalidCategorySlug
	}
	if strings.ContainsAny(slug, " ") {
		return ErrInvalidCategorySlug
	}
	return nil
}

func ValidateCategoryName(name string) error {
	if name == "" || len(name) > maxCategoryNameLength {
		return ErrInvalidCategoryName
	}
	return nil
}
