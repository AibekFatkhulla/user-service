package server

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"
	"user-service/internal/domain"
	"user-service/internal/service"

	log "github.com/sirupsen/logrus"

	"github.com/labstack/echo/v4"
)

type Server struct {
	userService service.UserServiceInterface
	db          *sql.DB
}

func NewServer(userService service.UserServiceInterface, db *sql.DB) *Server {
	return &Server{
		userService: userService,
		db:          db,
	}
}

func (s *Server) HealthCheck(c echo.Context) error {
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

func (s *Server) CreateUser(c echo.Context) error {
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
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, user)
}

func (s *Server) GetUser(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
	}

	ctx := c.Request().Context()
	user, err := s.userService.GetUser(ctx, id)
	if err != nil {
		if err.Error() == "user not found" || err.Error() == "failed to get user: user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
		}
		log.WithError(err).Error("Failed to get user")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, user)
}

func (s *Server) GetUserByEmail(c echo.Context) error {
	email := c.Param("email")
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "email is required",
		})
	}

	ctx := c.Request().Context()
	user, err := s.userService.GetUserByEmail(ctx, email)
	if err != nil {
		if err.Error() == "user not found" || err.Error() == "failed to get user by email: user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
		}
		log.WithError(err).Error("Failed to get user by email")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, user)
}

func (s *Server) UpdateUser(c echo.Context) error {
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
		if err.Error() == "user not found" || err.Error() == "failed to update user: user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
		}
		log.WithError(err).Error("Failed to update user")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, user)
}

func (s *Server) DeleteUser(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user ID is required",
		})
	}

	ctx := c.Request().Context()
	if err := s.userService.DeleteUser(ctx, id); err != nil {
		if err.Error() == "user not found" || err.Error() == "failed to delete user: user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
		}
		log.WithError(err).Error("Failed to delete user")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusNoContent, nil)
}

func (s *Server) ListUsers(c echo.Context) error {
	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")

	limit := 10 // Default
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
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, users)
}

// AddCoinsRequest - структура для запроса добавления монет
type AddCoinsRequest struct {
	Coins int64 `json:"coins"`
}

// SubscriptionRequest - структура для запроса подписки
type SubscriptionRequest struct {
	DurationHours int `json:"duration_hours"` // Продолжительность в часах
}

func (s *Server) AddCoins(c echo.Context) error {
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
		if err.Error() == "user not found" || err.Error() == "failed to add coins: user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
		}
		log.WithError(err).Error("Failed to add coins")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "coins added successfully",
	})
}

func (s *Server) ActivateSubscription(c echo.Context) error {
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
		if err.Error() == "user not found" || err.Error() == "failed to activate subscription: user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
		}
		log.WithError(err).Error("Failed to activate subscription")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "subscription activated successfully",
	})
}

func (s *Server) RenewSubscription(c echo.Context) error {
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
		if err.Error() == "user not found" || err.Error() == "failed to renew subscription: user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
		}
		if err.Error() == "user does not have an active subscription" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "user does not have an active subscription",
			})
		}
		log.WithError(err).Error("Failed to renew subscription")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "subscription renewed successfully",
	})
}