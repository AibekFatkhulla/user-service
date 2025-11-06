package server

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"
	"user-service/internal/domain"

	log "github.com/sirupsen/logrus"

	"github.com/labstack/echo/v4"
)

// UserService defines the interface for user business logic
type UserService interface {
	CreateUser(ctx context.Context, req domain.CreateUserRequest) (*domain.User, error)
	GetUser(ctx context.Context, id string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateUser(ctx context.Context, id string, req domain.UpdateUserRequest) (*domain.User, error)
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error)
	AddCoins(ctx context.Context, userID string, coins int64) error
	DeductCoins(ctx context.Context, userID string, coins int64) error
	ActivateSubscription(ctx context.Context, userID string, duration time.Duration) error
	RenewSubscription(ctx context.Context, userID string, duration time.Duration) error
	HasAccessByUser(user *domain.User) bool
}

type server struct {
	userService UserService
	db          *sql.DB
}

func NewServer(userService UserService, db *sql.DB) *server {
	return &server{
		userService: userService,
		db:          db,
	}
}

// handleError processes domain errors and returns appropriate HTTP response
func handleError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return http.StatusNotFound, "user not found"
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return http.StatusConflict, "user with this email already exists"
	case errors.Is(err, domain.ErrEmailRequired):
		return http.StatusBadRequest, "email is required"
	case errors.Is(err, domain.ErrNameRequired):
		return http.StatusBadRequest, "name is required"
	case errors.Is(err, domain.ErrUserIDRequired):
		return http.StatusBadRequest, "user ID is required"
	case errors.Is(err, domain.ErrInvalidEmailFormat):
		return http.StatusBadRequest, "invalid email format"
	case errors.Is(err, domain.ErrInvalidStatus):
		return http.StatusBadRequest, "invalid status"
	case errors.Is(err, domain.ErrInvalidCoinsAmount):
		return http.StatusBadRequest, "coins must be greater than 0"
	case errors.Is(err, domain.ErrInsufficientCoinsBalance):
		return http.StatusBadRequest, "insufficient coins balance"
	case errors.Is(err, domain.ErrInvalidSubscriptionDuration):
		return http.StatusBadRequest, "subscription duration must be greater than 0"
	case errors.Is(err, domain.ErrSubscriptionAlreadyActive):
		return http.StatusBadRequest, "subscription already active"
	case errors.Is(err, domain.ErrNoActiveSubscription):
		return http.StatusBadRequest, "user does not have an active subscription"
	case errors.Is(err, domain.ErrEmailTooLong):
		return http.StatusBadRequest, "email is too long"
	case errors.Is(err, domain.ErrNameTooLong):
		return http.StatusBadRequest, "name is too long"
	case errors.Is(err, domain.ErrInvalidUUID):
		return http.StatusBadRequest, "invalid user ID format"
	case errors.Is(err, domain.ErrCoinsAmountTooLarge):
		return http.StatusBadRequest, "coins amount is too large"
	case errors.Is(err, domain.ErrListLimitTooLarge):
		return http.StatusBadRequest, "list limit is too large"
	case errors.Is(err, domain.ErrListOffsetTooLarge):
		return http.StatusBadRequest, "list offset is too large"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

func (s *server) HealthCheck(c echo.Context) error {
	if err := s.db.Ping(); err != nil {
		log.WithField("error", err).Error("Health check failed: database is down")
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  "database connection error",
		})
	}
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

func (s *server) CreateUser(c echo.Context) error {
	var req domain.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	ctx := c.Request().Context()
	user, err := s.userService.CreateUser(ctx, req)
	if err != nil {
		log.WithError(err).Error("Failed to create user")
		statusCode, errorMsg := handleError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusCreated, user)
}

func (s *server) GetUser(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
	}

	ctx := c.Request().Context()
	user, err := s.userService.GetUser(ctx, id)
	if err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to get user")
		statusCode, errorMsg := handleError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	hasAccess := s.userService.HasAccessByUser(user)

	response := map[string]interface{}{
		"id":                    user.ID,
		"email":                 user.Email,
		"name":                  user.Name,
		"coins_balance":         user.CoinsBalance,
		"total_coins_purchased": user.TotalCoinsPurchased,
		"is_trial":              user.IsTrial,
		"trial_ends_at":         user.TrialEndsAt,
		"has_subscription":      user.HasSubscription,
		"subscription_ends_at":  user.SubscriptionEndsAt,
		"status":                user.Status,
		"created_at":            user.CreatedAt,
		"updated_at":            user.UpdatedAt,
		"has_access":            hasAccess,
	}

	return c.JSON(http.StatusOK, response)
}

func (s *server) GetUserByEmail(c echo.Context) error {
	email := c.Param("email")
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "email is required",
		})
	}

	ctx := c.Request().Context()
	user, err := s.userService.GetUserByEmail(ctx, email)
	if err != nil {
		log.WithError(err).WithField("email", email).Error("Failed to get user by email")
		statusCode, errorMsg := handleError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	hasAccess := s.userService.HasAccessByUser(user)

	response := map[string]interface{}{
		"id":                    user.ID,
		"email":                 user.Email,
		"name":                  user.Name,
		"coins_balance":         user.CoinsBalance,
		"total_coins_purchased": user.TotalCoinsPurchased,
		"is_trial":              user.IsTrial,
		"trial_ends_at":         user.TrialEndsAt,
		"has_subscription":      user.HasSubscription,
		"subscription_ends_at":  user.SubscriptionEndsAt,
		"status":                user.Status,
		"created_at":            user.CreatedAt,
		"updated_at":            user.UpdatedAt,
		"has_access":            hasAccess,
	}

	return c.JSON(http.StatusOK, response)
}

func (s *server) UpdateUser(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
	}

	var req domain.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	ctx := c.Request().Context()
	user, err := s.userService.UpdateUser(ctx, id, req)
	if err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to update user")
		statusCode, errorMsg := handleError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, user)
}

func (s *server) DeleteUser(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
	}

	ctx := c.Request().Context()
	if err := s.userService.DeleteUser(ctx, id); err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to delete user")
		statusCode, errorMsg := handleError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusNoContent, nil)
}

func (s *server) ListUsers(c echo.Context) error {
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

	ctx := c.Request().Context()
	users, err := s.userService.ListUsers(ctx, limit, offset)
	if err != nil {
		log.WithError(err).Error("Failed to list users")
		statusCode, errorMsg := handleError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, users)
}

// AddCoinsRequest - request structure to add coins
type AddCoinsRequest struct {
	Coins int64 `json:"coins"`
}

// SubscriptionRequest - request structure for subscription
type SubscriptionRequest struct {
	DurationHours int `json:"duration_hours"`
}

func (s *server) AddCoins(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
	}

	var req AddCoinsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	ctx := c.Request().Context()
	if err := s.userService.AddCoins(ctx, id, req.Coins); err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to add coins")
		statusCode, errorMsg := handleError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "coins added successfully",
	})
}

func (s *server) DeductCoins(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
	}

	var req AddCoinsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if req.Coins <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "coins must be greater than 0",
		})
	}

	ctx := c.Request().Context()
	if err := s.userService.DeductCoins(ctx, id, req.Coins); err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to deduct coins")
		statusCode, errorMsg := handleError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "coins deducted successfully",
	})
}

func (s *server) ActivateSubscription(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
	}

	var req SubscriptionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if req.DurationHours <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "duration_hours must be greater than 0",
		})
	}

	duration := time.Duration(req.DurationHours) * time.Hour

	ctx := c.Request().Context()
	if err := s.userService.ActivateSubscription(ctx, id, duration); err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to activate subscription")
		statusCode, errorMsg := handleError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "subscription activated successfully",
	})
}

func (s *server) RenewSubscription(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
	}

	var req SubscriptionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if req.DurationHours <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "duration_hours must be greater than 0",
		})
	}

	duration := time.Duration(req.DurationHours) * time.Hour

	ctx := c.Request().Context()
	if err := s.userService.RenewSubscription(ctx, id, duration); err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to renew subscription")
		statusCode, errorMsg := handleError(err)
		return c.JSON(statusCode, map[string]string{
			"error": errorMsg,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "subscription renewed successfully",
	})
}

func (s *server) HasAccess(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
	}

	ctx := c.Request().Context()
	user, err := s.userService.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
		}
		log.WithError(err).WithField("user_id", id).Error("Failed to get user")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "internal server error",
		})
	}

	hasAccess := s.userService.HasAccessByUser(user)

	return c.JSON(http.StatusOK, map[string]bool{
		"has_access": hasAccess,
	})
}
