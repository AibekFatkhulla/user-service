package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"user-service/internal/domain"

	log "github.com/sirupsen/logrus"
)

type postgresProductRepository struct {
	db *sql.DB
}

func NewPostgresProductRepository(db *sql.DB) *postgresProductRepository {
	return &postgresProductRepository{db: db}
}

func (r *postgresProductRepository) ListProducts(ctx context.Context, categoryID *string, onlyActive bool, limit, offset int) ([]domain.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var query strings.Builder
	args := []interface{}{}
	argPos := 1

	query.WriteString(`SELECT id, category_id, slug, name, description, price_coins, metadata, is_active, created_at, updated_at 
	                   FROM products 
	                   WHERE 1=1`)

	if categoryID != nil {
		query.WriteString(fmt.Sprintf(" AND category_id = $%d", argPos))
		args = append(args, *categoryID)
		argPos++
	}

	if onlyActive {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argPos))
		args = append(args, true)
		argPos++
	}

	query.WriteString(" ORDER BY created_at DESC")
	query.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1))
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var product domain.Product
		var metadata sql.NullString
		err := rows.Scan(
			&product.ID,
			&product.CategoryID,
			&product.Slug,
			&product.Name,
			&product.Description,
			&product.PriceCoins,
			&metadata,
			&product.IsActive,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			log.WithError(err).Error("Failed to scan product row")
			return nil, err
		}

		if metadata.Valid {
			product.Metadata = metadata.String
		}

		products = append(products, product)
	}

	return products, rows.Err()
}

func (r *postgresProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var product domain.Product
	var metadata sql.NullString
	query := `SELECT id, category_id, slug, name, description, price_coins, metadata, is_active, created_at, updated_at 
	          FROM products 
	          WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.CategoryID,
		&product.Slug,
		&product.Name,
		&product.Description,
		&product.PriceCoins,
		&metadata,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		log.WithError(err).WithField("product_id", id).Error("Failed to get product by ID")
		return nil, err
	}

	if metadata.Valid {
		product.Metadata = metadata.String
	}

	return &product, nil
}

func (r *postgresProductRepository) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var product domain.Product
	var metadata sql.NullString
	query := `SELECT id, category_id, slug, name, description, price_coins, metadata, is_active, created_at, updated_at 
	          FROM products 
	          WHERE slug = $1`

	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&product.ID,
		&product.CategoryID,
		&product.Slug,
		&product.Name,
		&product.Description,
		&product.PriceCoins,
		&metadata,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		log.WithError(err).WithField("slug", slug).Error("Failed to get product by slug")
		return nil, err
	}

	if metadata.Valid {
		product.Metadata = metadata.String
	}

	return &product, nil
}

func (r *postgresProductRepository) Create(ctx context.Context, req domain.CreateProductRequest) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"slug":        req.Slug,
		"name":        req.Name,
		"category_id": req.CategoryID,
	}).Info("Creating new product")

	query := `INSERT INTO products (category_id, slug, name, description, price_coins, metadata, is_active)
	          VALUES ($1, $2, $3, $4, $5, $6, $7)
	          RETURNING id, category_id, slug, name, description, price_coins, metadata, is_active, created_at, updated_at`

	var product domain.Product
	var metadata sql.NullString
	
	var metadataValue interface{}
	if req.Metadata != "" {
		metadataValue = req.Metadata
	} else {
		metadataValue = nil
	}
	
	err := r.db.QueryRowContext(ctx, query,
		req.CategoryID,
		req.Slug,
		req.Name,
		req.Description,
		req.PriceCoins,
		metadataValue,
		req.IsActive,
	).Scan(
		&product.ID,
		&product.CategoryID,
		&product.Slug,
		&product.Name,
		&product.Description,
		&product.PriceCoins,
		&metadata,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"slug":        req.Slug,
			"name":        req.Name,
			"category_id": req.CategoryID,
		}).Error("Failed to create product")
		return nil, err
	}

	if metadata.Valid {
		product.Metadata = metadata.String
	}

	return &product, nil
}

func (r *postgresProductRepository) Update(ctx context.Context, id string, req domain.UpdateProductRequest) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	setParts := []string{}
	args := []interface{}{}
	argPos := 1

	if req.CategoryID != nil {
		setParts = append(setParts, fmt.Sprintf("category_id = $%d", argPos))
		args = append(args, *req.CategoryID)
		argPos++
	}
	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *req.Name)
		argPos++
	}
	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *req.Description)
		argPos++
	}
	if req.PriceCoins != nil {
		setParts = append(setParts, fmt.Sprintf("price_coins = $%d", argPos))
		args = append(args, *req.PriceCoins)
		argPos++
	}
	if req.Metadata != nil {
		setParts = append(setParts, fmt.Sprintf("metadata = $%d", argPos))
		args = append(args, *req.Metadata)
		argPos++
	}
	if req.IsActive != nil {
		setParts = append(setParts, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *req.IsActive)
		argPos++
	}

	if len(setParts) == 0 {
		return r.GetByID(ctx, id)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(`UPDATE products 
	                      SET %s 
	                      WHERE id = $%d 
	                      RETURNING id, category_id, slug, name, description, price_coins, metadata, is_active, created_at, updated_at`,
		strings.Join(setParts, ", "), argPos)

	var product domain.Product
	var metadata sql.NullString
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&product.ID,
		&product.CategoryID,
		&product.Slug,
		&product.Name,
		&product.Description,
		&product.PriceCoins,
		&metadata,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrProductNotFound
	}
	if err != nil {
		log.WithError(err).WithField("product_id", id).Error("Failed to update product")
		return nil, err
	}

	if metadata.Valid {
		product.Metadata = metadata.String
	}

	return &product, nil
}

func (r *postgresProductRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithField("product_id", id).Info("Deleting product")

	query := `DELETE FROM products WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		log.WithError(err).WithField("product_id", id).Error("Failed to delete product")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrProductNotFound
	}

	return nil
}