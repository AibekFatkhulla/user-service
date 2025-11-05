package service

import (
	"context"
	"fmt"
	"regexp"
	"time"
	"user-service/internal/domain"
	"user-service/internal/repository"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type UserServiceInterface interface {
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

type UserService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) *UserService {
	return &UserService{userRepository: userRepository}
}

// ValidateStatus validates user status
func ValidateStatus(status string) error {
	validStatuses := domain.ValidStatuses()
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return nil
		}
	}
	return fmt.Errorf("invalid status: %s. Valid statuses: %v", status, validStatuses)
}

func (s *UserService) CreateUser(ctx context.Context, req domain.CreateUserRequest) (*domain.User, error) {
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return nil, fmt.Errorf("invalid email format")
	}

	existingUserByEmail, err := s.userRepository.GetByEmail(ctx, req.Email)
	if err == nil && existingUserByEmail != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	userID := uuid.New().String()

	trialEndsAt := time.Now().Add(3 * 24 * time.Hour) // 3 days

	user := &domain.User{
		ID:                  userID,
		Email:               req.Email,
		Name:                req.Name,
		CoinsBalance:        200,
		TotalCoinsPurchased: 0,
		IsTrial:             true,
		TrialEndsAt:         &trialEndsAt,
		HasSubscription:     false,
		SubscriptionEndsAt:  nil,
		Status:              domain.StatusActive,
	}

	if err := s.userRepository.Create(ctx, user); err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to create user")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	log.WithFields(log.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("User successfully created")

	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, id string) (*domain.User, error) {
	if id == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	user, err := s.userRepository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	user, err := s.userRepository.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id string, req domain.UpdateUserRequest) (*domain.User, error) {
	if id == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	user, err := s.userRepository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if req.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(req.Email) {
			return nil, fmt.Errorf("invalid email format")
		}
		if req.Email != user.Email {
			existingUser, err := s.userRepository.GetByEmail(ctx, req.Email)
			if err == nil && existingUser != nil {
				return nil, fmt.Errorf("user with email %s already exists", req.Email)
			}
		}
		if err := s.userRepository.UpdateEmail(ctx, id, req.Email); err != nil {
			log.WithError(err).WithField("user_id", id).Error("Failed to update user email")
			return nil, fmt.Errorf("failed to update email: %w", err)
		}
		user.Email = req.Email
	}

	if req.Name != "" {
		if err := s.userRepository.UpdateName(ctx, id, req.Name); err != nil {
			log.WithError(err).WithField("user_id", id).Error("Failed to update user name")
			return nil, fmt.Errorf("failed to update name: %w", err)
		}
		user.Name = req.Name
	}

	if req.Status != nil {
		if err := ValidateStatus(*req.Status); err != nil {
			return nil, err
		}
		if err := s.userRepository.UpdateStatus(ctx, id, *req.Status); err != nil {
			log.WithError(err).WithField("user_id", id).Error("Failed to update user status")
			return nil, fmt.Errorf("failed to update status: %w", err)
		}
		user.Status = *req.Status
	}

	log.WithField("user_id", id).Info("User successfully updated")
	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("user ID is required")
	}

	if err := s.userRepository.Delete(ctx, id); err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	log.WithField("user_id", id).Info("User successfully deleted")
	return nil
}

func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	users, err := s.userRepository.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

func (s *UserService) AddCoins(ctx context.Context, userID string, coins int64) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if coins <= 0 {
		return fmt.Errorf("coins must be greater than 0")
	}

	if err := s.userRepository.AddCoinsAtomic(ctx, userID, coins); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"user_id": userID,
			"coins":   coins,
		}).Error("Failed to add coins to user")
		return err
	}

	log.WithFields(log.Fields{
		"user_id":     userID,
		"coins_added": coins,
	}).Info("Coins successfully added to user")

	return nil
}

func (s *UserService) DeductCoins(ctx context.Context, userID string, coins int64) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if coins <= 0 {
		return fmt.Errorf("coins must be greater than 0")
	}

	if err := s.userRepository.DeductCoinsAtomic(ctx, userID, coins); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"user_id": userID,
			"coins":   coins,
		}).Error("Failed to deduct coins from user")
		return err
	}

	log.WithFields(log.Fields{
		"user_id":        userID,
		"coins_deducted": coins,
	}).Info("Coins successfully deducted from user")

	return nil
}

func (s *UserService) ActivateSubscription(ctx context.Context, userID string, duration time.Duration) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if duration <= 0 {
		return fmt.Errorf("subscription duration must be greater than 0")
	}

	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	subscriptionEndsAt := time.Now().Add(duration)
	isTrial := false

	if err := s.userRepository.AddCoinsAtomic(ctx, userID, 5000); err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to add coins for subscription")
		return fmt.Errorf("failed to add coins: %w", err)
	}

	if err := s.userRepository.ActivateSubscriptionAtomic(ctx, userID, isTrial, user.TrialEndsAt, &subscriptionEndsAt); err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to activate subscription")
		if err.Error() == "subscription already active" {
			return fmt.Errorf("subscription already active")
		}
		return fmt.Errorf("failed to activate subscription: %w", err)
	}

	log.WithFields(log.Fields{
		"user_id":              userID,
		"coins_added":          5000,
		"subscription_ends_at": subscriptionEndsAt,
	}).Info("Subscription successfully activated")

	return nil
}

func (s *UserService) RenewSubscription(ctx context.Context, userID string, duration time.Duration) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if duration <= 0 {
		return fmt.Errorf("subscription duration must be greater than 0")
	}

	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	var newEndsAt time.Time
	if user.SubscriptionEndsAt != nil && user.SubscriptionEndsAt.After(time.Now()) {
		newEndsAt = user.SubscriptionEndsAt.Add(duration)
	} else {
		newEndsAt = time.Now().Add(duration)
	}

	if err := s.userRepository.AddCoinsAtomic(ctx, userID, 5000); err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to add coins for subscription")
		return fmt.Errorf("failed to add coins: %w", err)
	}

	if err := s.userRepository.RenewSubscriptionAtomic(ctx, userID, &newEndsAt); err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to renew subscription")
		return fmt.Errorf("failed to renew subscription: %w", err)
	}

	log.WithFields(log.Fields{
		"user_id":              userID,
		"coins_added":          5000,
		"subscription_ends_at": newEndsAt,
	}).Info("Subscription successfully renewed")

	return nil
}

// HasAccessByUser checks if user has access to functionality
// Access is granted if:
// 1. status == "active"
// 2. AND (has active subscription OR trial is active)
func (s *UserService) HasAccessByUser(user *domain.User) bool {
	if user == nil {
		return false
	}

	if user.Status != domain.StatusActive {
		return false
	}

	now := time.Now()

	if user.HasSubscription && user.SubscriptionEndsAt != nil {
		if user.SubscriptionEndsAt.After(now) || user.SubscriptionEndsAt.Equal(now) {
			return true
		}
	}

	if user.IsTrial && user.TrialEndsAt != nil {
		if user.TrialEndsAt.After(now) || user.TrialEndsAt.Equal(now) {
			return true
		}
	}

	return false
}
