package service

import (
	"context"
	"fmt"
	"regexp"
	"time"
	"user-service/internal/domain"
	"user-service/internal/repository"

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
	ActivateSubscription(ctx context.Context, userID string, duration time.Duration) error
	RenewSubscription(ctx context.Context, userID string, duration time.Duration) error
}

type UserService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) *UserService {
	return &UserService{userRepository: userRepository}
}

func (s *UserService) CreateUser(ctx context.Context, req domain.CreateUserRequest) (*domain.User, error) {
	// Валидация входных данных
	if req.ID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Простая валидация email (можно использовать regex из notification-service)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(req.Email) {
		return nil, fmt.Errorf("invalid email format")
	}

	// Проверка существования пользователя по ID
	existingUser, err := s.userRepository.GetByID(ctx, req.ID)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with ID %s already exists", req.ID)
	}

	// Проверка существования пользователя по Email
	existingUserByEmail, err := s.userRepository.GetByEmail(ctx, req.Email)
	if err == nil && existingUserByEmail != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Создание пользователя с начальными значениями
	trialEndsAt := time.Now().Add(3 * 24 * time.Hour) // 3 дня

	user := &domain.User{
		ID:                  req.ID,
		Email:               req.Email,
		Name:                req.Name,
		CoinsBalance:        200, // Триал монеты
		TotalCoinsPurchased: 0,   // Триал монеты не учитываются
		IsTrial:             true,
		TrialEndsAt:         &trialEndsAt,
		HasSubscription:     false,
		SubscriptionEndsAt:  nil,
		Status:              "active",
	}

	if err := s.userRepository.Create(ctx, user); err != nil {
		log.WithError(err).WithField("user_id", req.ID).Error("Failed to create user")
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
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	user, err := s.userRepository.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id string, req domain.UpdateUserRequest) (*domain.User, error) {
	if id == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Получаем текущего пользователя
	user, err := s.userRepository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Обновляем только переданные поля
	if req.Email != "" {
		// Валидация email
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(req.Email) {
			return nil, fmt.Errorf("invalid email format")
		}
		// Проверка уникальности email (если изменился)
		if req.Email != user.Email {
			existingUser, err := s.userRepository.GetByEmail(ctx, req.Email)
			if err == nil && existingUser != nil {
				return nil, fmt.Errorf("user with email %s already exists", req.Email)
			}
		}
		user.Email = req.Email
	}

	if req.Name != "" {
		user.Name = req.Name
	}

	// Сохраняем обновленного пользователя
	if err := s.userRepository.Update(ctx, user); err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to update user")
		return nil, fmt.Errorf("failed to update user: %w", err)
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
		limit = 10 // Дефолтное значение
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

	// Получаем пользователя
	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Добавляем монеты к балансу (без ограничений)
	user.CoinsBalance += coins
	user.TotalCoinsPurchased += coins

	// Сохраняем обновленного пользователя
	if err := s.userRepository.Update(ctx, user); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"user_id": userID,
			"coins":   coins,
		}).Error("Failed to add coins to user")
		return fmt.Errorf("failed to add coins: %w", err)
	}

	log.WithFields(log.Fields{
		"user_id":           userID,
		"coins_added":       coins,
		"new_balance":       user.CoinsBalance,
		"total_purchased":   user.TotalCoinsPurchased,
	}).Info("Coins successfully added to user")

	return nil
}

func (s *UserService) ActivateSubscription(ctx context.Context, userID string, duration time.Duration) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if duration <= 0 {
		return fmt.Errorf("subscription duration must be greater than 0")
	}

	// Получаем пользователя
	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Добавляем 5000 монет при оформлении подписки
	user.CoinsBalance += 5000
	user.TotalCoinsPurchased += 5000

	// Устанавливаем подписку
	subscriptionEndsAt := time.Now().Add(duration)
	user.HasSubscription = true
	user.SubscriptionEndsAt = &subscriptionEndsAt
	user.IsTrial = false // Отключаем триал при оформлении подписки

	// Сохраняем обновленного пользователя
	if err := s.userRepository.Update(ctx, user); err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to activate subscription")
		return fmt.Errorf("failed to activate subscription: %w", err)
	}

	log.WithFields(log.Fields{
		"user_id":             userID,
		"coins_added":         5000,
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

	// Получаем пользователя
	user, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Проверяем, что у пользователя есть подписка
	if !user.HasSubscription {
		return fmt.Errorf("user does not have an active subscription")
	}

	// Добавляем 5000 монет при продлении подписки
	user.CoinsBalance += 5000
	user.TotalCoinsPurchased += 5000

	// Продлеваем подписку
	var newEndsAt time.Time
	if user.SubscriptionEndsAt != nil && user.SubscriptionEndsAt.After(time.Now()) {
		// Если подписка еще активна, продлеваем от текущей даты окончания
		newEndsAt = user.SubscriptionEndsAt.Add(duration)
	} else {
		// Если подписка истекла, продлеваем от текущей даты
		newEndsAt = time.Now().Add(duration)
	}
	user.SubscriptionEndsAt = &newEndsAt

	// Сохраняем обновленного пользователя
	if err := s.userRepository.Update(ctx, user); err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to renew subscription")
		return fmt.Errorf("failed to renew subscription: %w", err)
	}

	log.WithFields(log.Fields{
		"user_id":             userID,
		"coins_added":         5000,
		"subscription_ends_at": newEndsAt,
	}).Info("Subscription successfully renewed")

	return nil
}