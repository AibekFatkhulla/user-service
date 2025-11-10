package service

import (
	"context"
	"user-service/internal/domain"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type ProductRepository interface {
	ListProducts(ctx context.Context, categoryID *string, onlyActive bool, limit, offset int) ([]domain.Product, error)
	GetByID(ctx context.Context, id string) (*domain.Product, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Product, error)
	Create(ctx context.Context, req domain.CreateProductRequest) (*domain.Product, error)
	Update(ctx context.Context, id string, req domain.UpdateProductRequest) (*domain.Product, error)
	Delete(ctx context.Context, id string) error
}

type productService struct {
	productRepo ProductRepository
}

func NewProductService(productRepo ProductRepository) *productService {
	return &productService{
		productRepo: productRepo,
	}
}

func (s *productService) ListProducts(ctx context.Context, categoryID *string, onlyActive bool, limit, offset int) ([]domain.Product, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > domain.MaxListLimit {
		limit = domain.MaxListLimit
	}
	if offset < 0 {
		offset = 0
	}

	products, err := s.productRepo.ListProducts(ctx, categoryID, onlyActive, limit, offset)
	if err != nil {
		log.WithError(err).Error("Failed to list products")
		return nil, err
	}
	return products, nil
}

func (s *productService) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	if id == "" {
		return nil, domain.ErrInvalidUUID
	}

	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (s *productService) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	if err := domain.ValidateProductSlug(slug); err != nil {
		return nil, err
	}

	product, err := s.productRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (s *productService) CreateProduct(ctx context.Context, req domain.CreateProductRequest) (*domain.Product, error) {
	if req.CategoryID == "" {
		return nil, domain.ErrInvalidUUID
	}
	if _, err := uuid.Parse(req.CategoryID); err != nil {
		return nil, domain.ErrInvalidUUID
	}
	if err := domain.ValidateProductSlug(req.Slug); err != nil {
		return nil, err
	}
	if err := domain.ValidateProductName(req.Name); err != nil {
		return nil, err
	}
	if err := domain.ValidateProductPrice(req.PriceCoins); err != nil {
		return nil, err
	}

	existing, err := s.productRepo.GetBySlug(ctx, req.Slug)
	if err != nil && err != domain.ErrProductNotFound {
		log.WithError(err).WithField("slug", req.Slug).Error("Failed to check product existence")
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrProductSlugExists
	}

	product, err := s.productRepo.Create(ctx, req)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"slug":        req.Slug,
			"name":        req.Name,
			"category_id": req.CategoryID,
		}).Error("Failed to create product")
		return nil, err
	}

	return product, nil
}

func (s *productService) UpdateProduct(ctx context.Context, id string, req domain.UpdateProductRequest) (*domain.Product, error) {
	if id == "" {
		return nil, domain.ErrInvalidUUID
	}

	if req.Name != nil {
		if err := domain.ValidateProductName(*req.Name); err != nil {
			return nil, err
		}
	}
	if req.PriceCoins != nil {
		if err := domain.ValidateProductPrice(*req.PriceCoins); err != nil {
			return nil, err
		}
	}

	product, err := s.productRepo.Update(ctx, id, req)
	if err != nil {
		log.WithError(err).WithField("product_id", id).Error("Failed to update product")
		return nil, err
	}

	return product, nil
}

func (s *productService) DeleteProduct(ctx context.Context, id string) error {
	if id == "" {
		return domain.ErrInvalidUUID
	}

	err := s.productRepo.Delete(ctx, id)
	if err != nil {
		log.WithError(err).WithField("product_id", id).Error("Failed to delete product")
		return err
	}

	return nil
}