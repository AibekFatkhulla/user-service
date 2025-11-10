package service

import (
	"context"
	"user-service/internal/domain"

	log "github.com/sirupsen/logrus"
)

type ProductCategoryRepository interface {
	ListCategories(ctx context.Context, onlyActive bool) ([]domain.ProductCategory, error)
	GetByID(ctx context.Context, id string) (*domain.ProductCategory, error)
	GetBySlug(ctx context.Context, slug string) (*domain.ProductCategory, error)
	Create(ctx context.Context, req domain.CreateCategoryRequest) (*domain.ProductCategory, error)
	Update(ctx context.Context, id string, req domain.UpdateCategoryRequest) (*domain.ProductCategory, error)
	Delete(ctx context.Context, id string) error
}

type productCategoryService struct {
	categoryRepo ProductCategoryRepository
}

func NewProductCategoryService(categoryRepo ProductCategoryRepository) *productCategoryService {
	return &productCategoryService{
		categoryRepo: categoryRepo,
	}
}

func (s *productCategoryService) ListCategories(ctx context.Context, onlyActive bool) ([]domain.ProductCategory, error) {
	categories, err := s.categoryRepo.ListCategories(ctx, onlyActive)
	if err != nil {
		log.WithError(err).Error("Failed to list product categories")
		return nil, err
	}
	return categories, nil
}

func (s *productCategoryService) GetCategoryByID(ctx context.Context, id string) (*domain.ProductCategory, error) {
	if id == "" {
		return nil, domain.ErrInvalidUUID
	}

	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (s *productCategoryService) GetCategoryBySlug(ctx context.Context, slug string) (*domain.ProductCategory, error) {
	if err := domain.ValidateCategorySlug(slug); err != nil {
		return nil, err
	}

	category, err := s.categoryRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (s *productCategoryService) CreateCategory(ctx context.Context, req domain.CreateCategoryRequest) (*domain.ProductCategory, error) {
	if err := domain.ValidateCategorySlug(req.Slug); err != nil {
		return nil, err
	}
	if err := domain.ValidateCategoryName(req.Name); err != nil {
		return nil, err
	}

	existing, err := s.categoryRepo.GetBySlug(ctx, req.Slug)
	if err != nil && err != domain.ErrCategoryNotFound {
		log.WithError(err).WithField("slug", req.Slug).Error("Failed to check category existence")
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrCategorySlugExists
	}

	category, err := s.categoryRepo.Create(ctx, req)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"slug": req.Slug,
			"name": req.Name,
		}).Error("Failed to create product category")
		return nil, err
	}

	return category, nil
}

func (s *productCategoryService) UpdateCategory(ctx context.Context, id string, req domain.UpdateCategoryRequest) (*domain.ProductCategory, error) {
	if id == "" {
		return nil, domain.ErrInvalidUUID
	}

	if req.Name != nil {
		if err := domain.ValidateCategoryName(*req.Name); err != nil {
			return nil, err
		}
	}

	category, err := s.categoryRepo.Update(ctx, id, req)
	if err != nil {
		log.WithError(err).WithField("category_id", id).Error("Failed to update product category")
		return nil, err
	}

	return category, nil
}

func (s *productCategoryService) DeleteCategory(ctx context.Context, id string) error {
	if id == "" {
		return domain.ErrInvalidUUID
	}

	err := s.categoryRepo.Delete(ctx, id)
	if err != nil {
		log.WithError(err).WithField("category_id", id).Error("Failed to delete product category")
		return err
	}

	return nil
}