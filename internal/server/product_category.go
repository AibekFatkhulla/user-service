package server

import (
	"context"
	"errors"
	"net/http"
	"user-service/internal/domain"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

type ProductCategoryService interface {
	ListCategories(ctx context.Context, onlyActive bool) ([]domain.ProductCategory, error)
	GetCategoryByID(ctx context.Context, id string) (*domain.ProductCategory, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*domain.ProductCategory, error)
	CreateCategory(ctx context.Context, req domain.CreateCategoryRequest) (*domain.ProductCategory, error)
	UpdateCategory(ctx context.Context, id string, req domain.UpdateCategoryRequest) (*domain.ProductCategory, error)
	DeleteCategory(ctx context.Context, id string) error
}

type productCategoryServer struct {
	categoryService ProductCategoryService
}

func NewProductCategoryServer(categoryService ProductCategoryService) *productCategoryServer {
	return &productCategoryServer{
		categoryService: categoryService,
	}
}

func handleCategoryError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrCategoryNotFound):
		return http.StatusNotFound, "category not found"
	case errors.Is(err, domain.ErrCategorySlugExists):
		return http.StatusConflict, "category with this slug already exists"
	case errors.Is(err, domain.ErrInvalidCategorySlug), errors.Is(err, domain.ErrInvalidCategoryName), errors.Is(err, domain.ErrInvalidUUID):
		return http.StatusBadRequest, "invalid request"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

func (s *productCategoryServer) ListCategories(c echo.Context) error {
	onlyActive := c.QueryParam("only_active") == "true"

	categories, err := s.categoryService.ListCategories(c.Request().Context(), onlyActive)
	if err != nil {
		log.WithError(err).Error("Failed to list categories")
		statusCode, errorMsg := handleCategoryError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, categories)
}

func (s *productCategoryServer) GetCategoryByID(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	category, err := s.categoryService.GetCategoryByID(c.Request().Context(), id)
	if err != nil {
		log.WithError(err).WithField("category_id", id).Error("Failed to get category")
		statusCode, errorMsg := handleCategoryError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, category)
}

func (s *productCategoryServer) GetCategoryBySlug(c echo.Context) error {
	slug := c.Param("slug")
	if slug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	category, err := s.categoryService.GetCategoryBySlug(c.Request().Context(), slug)
	if err != nil {
		log.WithError(err).WithField("slug", slug).Error("Failed to get category by slug")
		statusCode, errorMsg := handleCategoryError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, category)
}

func (s *productCategoryServer) CreateCategory(c echo.Context) error {
	var req domain.CreateCategoryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	category, err := s.categoryService.CreateCategory(c.Request().Context(), req)
	if err != nil {
		log.WithError(err).Error("Failed to create category")
		statusCode, errorMsg := handleCategoryError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusCreated, category)
}

func (s *productCategoryServer) UpdateCategory(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	var req domain.UpdateCategoryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	category, err := s.categoryService.UpdateCategory(c.Request().Context(), id, req)
	if err != nil {
		log.WithError(err).WithField("category_id", id).Error("Failed to update category")
		statusCode, errorMsg := handleCategoryError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, category)
}

func (s *productCategoryServer) DeleteCategory(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	err := s.categoryService.DeleteCategory(c.Request().Context(), id)
	if err != nil {
		log.WithError(err).WithField("category_id", id).Error("Failed to delete category")
		statusCode, errorMsg := handleCategoryError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.NoContent(http.StatusNoContent)
}