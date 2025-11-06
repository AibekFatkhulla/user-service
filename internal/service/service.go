package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"
	"user-service/internal/domain"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, userID string, fields *domain.UpdateUserFields) error
	AddCoinsAtomic(ctx context.Context, userID string, coins int64) error
	DeductCoinsAtomic(ctx context.Context, userID string, coins int64) error
	ActivateSubscriptionAtomic(ctx context.Context, userID string, isTrial bool, trialEndsAt *time.Time, subscriptionEndsAt *time.Time) error
	RenewSubscriptionAtomic(ctx context.Context, userID string, subscriptionEndsAt *time.Time) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]domain.User, error)
}

type userService struct {
	userRepository UserRepository
}

func NewUserService(userRepository UserRepository) *userService {
	return &userService{userRepository: userRepository}
}

// ValidateStatus validates user status
func ValidateStatus(status string) error {
	validStatuses := domain.ValidStatuses()
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return nil
		}
	}
	return domain.ErrInvalidStatus
}

func (s *userService) CreateUser(ctx context.Context, req domain.CreateUserRequest) (*domain.User, error) {
	if req.Email == "" {
		return nil, domain.ErrEmailRequired
	}
	if len(req.Email) > domain.MaxEmailLength {
		return nil, domain.ErrEmailTooLong
	}
	if req.Name == "" {
		return nil, domain.ErrNameRequired
	}
	if len(req.Name) > domain.MaxNameLength {
		return nil, domain.ErrNameTooLong
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return nil, domain.ErrInvalidEmailFormat
	}

	existingUserByEmail, err := s.userRepository.GetByEmail(ctx, req.Email)
	if err == nil && existingUserByEmail != nil {
		return nil, domain.ErrEmailAlreadyExists
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

func (s *userService) GetUser(ctx context.Context, id string) (*domain.User, error) {
	if id == "" {
		return nil, domain.ErrUserIDRequired
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, domain.ErrInvalidUUID
	}

	user, err := s.userRepository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if email == "" {
		return nil, domain.ErrEmailRequired
	}

	user, err := s.userRepository.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) UpdateUser(ctx context.Context, id string, req domain.UpdateUserRequest) (*domain.User, error) {
	if id == "" {
		return nil, domain.ErrUserIDRequired
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, domain.ErrInvalidUUID
	}

	user, err := s.userRepository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Build update fields structure - only include changed fields
	updateFields := &domain.UpdateUserFields{}

	// Validate and prepare email update
	if req.Email != "" && req.Email != user.Email {
		if len(req.Email) > domain.MaxEmailLength {
			return nil, domain.ErrEmailTooLong
		}
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(req.Email) {
			return nil, domain.ErrInvalidEmailFormat
		}
		existingUser, err := s.userRepository.GetByEmail(ctx, req.Email)
		if err == nil && existingUser != nil {
			return nil, domain.ErrEmailAlreadyExists
		}
		updateFields.Email = &req.Email
		user.Email = req.Email
	}

	// Prepare name update
	if req.Name != "" && req.Name != user.Name {
		if len(req.Name) > domain.MaxNameLength {
			return nil, domain.ErrNameTooLong
		}
		updateFields.Name = &req.Name
		user.Name = req.Name
	}

	// Validate and prepare status update
	if req.Status != nil && *req.Status != user.Status {
		if err := ValidateStatus(*req.Status); err != nil {
			return nil, err
		}
		updateFields.Status = req.Status
		user.Status = *req.Status
	}

	// If no fields changed, return current user
	if updateFields.Email == nil && updateFields.Name == nil && updateFields.Status == nil {
		log.WithField("user_id", id).Info("No fields changed, skipping update")
		return user, nil
	}

	// Update user in repository (single transaction, only changed fields)
	if err := s.userRepository.Update(ctx, id, updateFields); err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to update user")
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	log.WithField("user_id", id).Info("User successfully updated")
	return user, nil
}

func (s *userService) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		return domain.ErrUserIDRequired
	}
	if _, err := uuid.Parse(id); err != nil {
		return domain.ErrInvalidUUID
	}

	if err := s.userRepository.Delete(ctx, id); err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	log.WithField("user_id", id).Info("User successfully deleted")
	return nil
}

func (s *userService) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > domain.MaxListLimit {
		return nil, domain.ErrListLimitTooLarge
	}
	if offset < 0 {
		offset = 0
	}
	if offset > domain.MaxListOffset {
		return nil, domain.ErrListOffsetTooLarge
	}

	users, err := s.userRepository.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

func (s *userService) AddCoins(ctx context.Context, userID string, coins int64) error {
	if userID == "" {
		return domain.ErrUserIDRequired
	}
	if _, err := uuid.Parse(userID); err != nil {
		return domain.ErrInvalidUUID
	}
	if coins <= 0 {
		return domain.ErrInvalidCoinsAmount
	}
	if coins > domain.MaxCoinsAmount {
		return domain.ErrCoinsAmountTooLarge
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

func (s *userService) DeductCoins(ctx context.Context, userID string, coins int64) error {
	if userID == "" {
		return domain.ErrUserIDRequired
	}
	if _, err := uuid.Parse(userID); err != nil {
		return domain.ErrInvalidUUID
	}
	if coins <= 0 {
		return domain.ErrInvalidCoinsAmount
	}
	if coins > domain.MaxCoinsAmount {
		return domain.ErrCoinsAmountTooLarge
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

func (s *userService) ActivateSubscription(ctx context.Context, userID string, duration time.Duration) error {
	if userID == "" {
		return domain.ErrUserIDRequired
	}
	if _, err := uuid.Parse(userID); err != nil {
		return domain.ErrInvalidUUID
	}
	if duration <= 0 {
		return domain.ErrInvalidSubscriptionDuration
	}

	maxDuration := time.Duration(domain.MaxSubscriptionDurationHours) * time.Hour
	if duration > maxDuration {
		return domain.ErrSubscriptionDurationTooLong
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
		if errors.Is(err, domain.ErrSubscriptionAlreadyActive) {
			return domain.ErrSubscriptionAlreadyActive
		}
		log.WithError(err).WithField("user_id", userID).Error("Failed to activate subscription")
		return fmt.Errorf("failed to activate subscription: %w", err)
	}

	log.WithFields(log.Fields{
		"user_id":              userID,
		"coins_added":          5000,
		"subscription_ends_at": subscriptionEndsAt,
	}).Info("Subscription successfully activated")

	return nil
}

func (s *userService) RenewSubscription(ctx context.Context, userID string, duration time.Duration) error {
	if userID == "" {
		return domain.ErrUserIDRequired
	}
	if _, err := uuid.Parse(userID); err != nil {
		return domain.ErrInvalidUUID
	}
	if duration <= 0 {
		return domain.ErrInvalidSubscriptionDuration
	}

	maxDuration := time.Duration(domain.MaxSubscriptionDurationHours) * time.Hour
	if duration > maxDuration {
		return domain.ErrSubscriptionDurationTooLong
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
func (s *userService) HasAccessByUser(user *domain.User) bool {
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
