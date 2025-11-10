package repository

import (
	"context"
	"database/sql"
	"time"
	"user-service/internal/domain"
	"strings"

	log "github.com/sirupsen/logrus"

)

type postgresProductCategoryRepository struct {
	db *sql.DB
}

func NewPostgresProductCategoryRepository(db *sql.DB) *postgresProductCategoryRepository {
	return &postgresProductCategoryRepository{db: db}
}

func (r *postgresProductCategoryRepository) ListCategories(ctx context.Context, onlyActive bool) ([]domain.ProductCategory, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var query string
	if onlyActive {
		query = `SELECT id, slug, name, description, position, is_active, created_at, updated_at 
		         FROM product_categories 
		         WHERE is_active = true 
		         ORDER BY position ASC, created_at ASC`
	} else {
		query = `SELECT id, slug, name, description, position, is_active, created_at, updated_at 
		         FROM product_categories 
		         ORDER BY position ASC, created_at ASC`
	}

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.ProductCategory
	for rows.Next() {
		var cat domain.ProductCategory
		err := rows.Scan(
			&cat.ID,
			&cat.Slug,
			&cat.Name,
			&cat.Description,
			&cat.Position,
			&cat.IsActive,
			&cat.CreatedAt,
			&cat.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}

	return categories, rows.Err()
}

func (r *postgresProductCategoryRepository) GetByID(ctx context.Context, id string) (*domain.ProductCategory, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var cat domain.ProductCategory
	query := `SELECT id, slug, name, description, position, is_active, created_at, updated_at 
	          FROM product_categories 
	          WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&cat.ID,
		&cat.Slug,
		&cat.Name,
		&cat.Description,
		&cat.Position,
		&cat.IsActive,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrCategoryNotFound
	}
	if err != nil {
		log.WithError(err).WithField("category_id", id).Error("Failed to get product category by ID")
		return nil, err
	}

	return &cat, nil
}

func (r *postgresProductCategoryRepository) GetBySlug(ctx context.Context, slug string) (*domain.ProductCategory, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var cat domain.ProductCategory
	query := `SELECT id, slug, name, description, position, is_active, created_at, updated_at 
	          FROM product_categories 
	          WHERE slug = $1`

	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&cat.ID,
		&cat.Slug,
		&cat.Name,
		&cat.Description,
		&cat.Position,
		&cat.IsActive,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrCategoryNotFound
	}
	if err != nil {
		log.WithError(err).WithField("slug", slug).Error("Failed to get product category by slug")
		return nil, err
	}

	return &cat, nil
}

func (r *postgresProductCategoryRepository) Create(ctx context.Context, req domain.CreateCategoryRequest) (*domain.ProductCategory, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `INSERT INTO product_categories (slug, name, description, position, is_active)
	          VALUES ($1, $2, $3, $4, $5)
	          RETURNING id, slug, name, description, position, is_active, created_at, updated_at`

	var cat domain.ProductCategory
	err := r.db.QueryRowContext(ctx, query,
		req.Slug,
		req.Name,
		req.Description,
		req.Position,
		req.IsActive,
	).Scan(
		&cat.ID,
		&cat.Slug,
		&cat.Name,
		&cat.Description,
		&cat.Position,
		&cat.IsActive,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)

	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"slug": req.Slug,
			"name": req.Name,
		}).Error("Failed to create product category")
		return nil, err
	}

	return &cat, nil
}

func (r *postgresProductCategoryRepository) Update(ctx context.Context, id string, req domain.UpdateCategoryRequest) (*domain.ProductCategory, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	setParts := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Name != nil {
		setParts = append(setParts, "name = $"+string(rune('0'+argPos)))
		args = append(args, *req.Name)
		argPos++
	}
	if req.Description != nil {
		setParts = append(setParts, "description = $"+string(rune('0'+argPos)))
		args = append(args, *req.Description)
		argPos++
	}
	if req.Position != nil {
		setParts = append(setParts, "position = $"+string(rune('0'+argPos)))
		args = append(args, *req.Position)
		argPos++
	}
	if req.IsActive != nil {
		setParts = append(setParts, "is_active = $"+string(rune('0'+argPos)))
		args = append(args, *req.IsActive)
		argPos++
	}

	if len(setParts) == 0 {
		return r.GetByID(ctx, id)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id)

	query := `UPDATE product_categories 
	          SET ` + strings.Join(setParts, ", ") + `
	          WHERE id = $` + string(rune('0'+argPos)) + `
	          RETURNING id, slug, name, description, position, is_active, created_at, updated_at`

	var cat domain.ProductCategory
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&cat.ID,
		&cat.Slug,
		&cat.Name,
		&cat.Description,
		&cat.Position,
		&cat.IsActive,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrCategoryNotFound
	}
	if err != nil {
		log.WithError(err).WithField("category_id", id).Error("Failed to update product category")
		return nil, err
	}

	return &cat, nil
}

func (r *postgresProductCategoryRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `DELETE FROM product_categories WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	
	if err != nil {
		log.WithError(err).WithField("category_id", id).Error("Failed to delete product category")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrCategoryNotFound
	}

	return nil
}