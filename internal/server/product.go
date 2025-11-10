package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"user-service/internal/domain"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

type ProductService interface {
	ListProducts(ctx context.Context, categoryID *string, onlyActive bool, limit, offset int) ([]domain.Product, error)
	GetProductByID(ctx context.Context, id string) (*domain.Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error)
	CreateProduct(ctx context.Context, req domain.CreateProductRequest) (*domain.Product, error)
	UpdateProduct(ctx context.Context, id string, req domain.UpdateProductRequest) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id string) error
}

type productServer struct {
	productService ProductService
}

func NewProductServer(productService ProductService) *productServer {
	return &productServer{
		productService: productService,
	}
}

func handleProductError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrProductNotFound):
		return http.StatusNotFound, "product not found"
	case errors.Is(err, domain.ErrProductSlugExists):
		return http.StatusConflict, "product with this slug already exists"
	case errors.Is(err, domain.ErrInvalidProductSlug), errors.Is(err, domain.ErrInvalidProductName), errors.Is(err, domain.ErrInvalidPrice), errors.Is(err, domain.ErrInvalidUUID):
		return http.StatusBadRequest, "invalid request"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

func (s *productServer) ListProducts(c echo.Context) error {
	categoryID := c.QueryParam("category_id")
	onlyActive := c.QueryParam("only_active") == "true"
	
	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")
	
	limit := 10
	offset := 0
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var categoryIDPtr *string
	if categoryID != "" {
		categoryIDPtr = &categoryID
	}

	products, err := s.productService.ListProducts(c.Request().Context(), categoryIDPtr, onlyActive, limit, offset)
	if err != nil {
		log.WithError(err).Error("Failed to list products")
		statusCode, errorMsg := handleProductError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, products)
}

func (s *productServer) GetProductByID(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	product, err := s.productService.GetProductByID(c.Request().Context(), id)
	if err != nil {
		log.WithError(err).WithField("product_id", id).Error("Failed to get product")
		statusCode, errorMsg := handleProductError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, product)
}

func (s *productServer) GetProductBySlug(c echo.Context) error {
	slug := c.Param("slug")
	if slug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	product, err := s.productService.GetProductBySlug(c.Request().Context(), slug)
	if err != nil {
		log.WithError(err).WithField("slug", slug).Error("Failed to get product by slug")
		statusCode, errorMsg := handleProductError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, product)
}

func (s *productServer) CreateProduct(c echo.Context) error {
	var req domain.CreateProductRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	product, err := s.productService.CreateProduct(c.Request().Context(), req)
	if err != nil {
		log.WithError(err).Error("Failed to create product")
		statusCode, errorMsg := handleProductError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusCreated, product)
}

func (s *productServer) UpdateProduct(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	var req domain.UpdateProductRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	product, err := s.productService.UpdateProduct(c.Request().Context(), id, req)
	if err != nil {
		log.WithError(err).WithField("product_id", id).Error("Failed to update product")
		statusCode, errorMsg := handleProductError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, product)
}

func (s *productServer) DeleteProduct(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}

	err := s.productService.DeleteProduct(c.Request().Context(), id)
	if err != nil {
		log.WithError(err).WithField("product_id", id).Error("Failed to delete product")
		statusCode, errorMsg := handleProductError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.NoContent(http.StatusNoContent)
}