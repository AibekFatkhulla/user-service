package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"user-service/internal/domain"

	log "github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateEmail(ctx context.Context, userID string, email string) error
	UpdateName(ctx context.Context, userID string, name string) error
	UpdateStatus(ctx context.Context, userID string, status string) error
	AddCoinsAtomic(ctx context.Context, userID string, coins int64) error
	DeductCoinsAtomic(ctx context.Context, userID string, coins int64) error
	ActivateSubscriptionAtomic(ctx context.Context, userID string, isTrial bool, trialEndsAt *time.Time, subscriptionEndsAt *time.Time) error
	RenewSubscriptionAtomic(ctx context.Context, userID string, subscriptionEndsAt *time.Time) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]domain.User, error)
}

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
	}).Info("Creating new user in database")

	query := `
		INSERT INTO users (
			id, email, name,
			coins_balance, total_coins_purchased,
			is_trial, trial_ends_at,
			has_subscription, subscription_ends_at,
			status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.CoinsBalance,
		user.TotalCoinsPurchased,
		user.IsTrial,
		user.TrialEndsAt,
		user.HasSubscription,
		user.SubscriptionEndsAt,
		user.Status,
	)

	if err != nil {
		log.WithError(err).WithField("user_id", user.ID).Error("Failed to create user")
		return fmt.Errorf("failed to create user: %w", err)
	}

	log.WithField("user_id", user.ID).Info("User successfully created")
	return nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, email, name,
			coins_balance, total_coins_purchased,
			is_trial, trial_ends_at,
			has_subscription, subscription_ends_at,
			status, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	var trialEndsAt, subscriptionEndsAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.CoinsBalance,
		&user.TotalCoinsPurchased,
		&user.IsTrial,
		&trialEndsAt,
		&user.HasSubscription,
		&subscriptionEndsAt,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		log.WithError(err).WithField("user_id", id).Error("Failed to get user by ID")
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	if trialEndsAt.Valid {
		user.TrialEndsAt = &trialEndsAt.Time
	}
	if subscriptionEndsAt.Valid {
		user.SubscriptionEndsAt = &subscriptionEndsAt.Time
	}

	return &user, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, email, name,
			coins_balance, total_coins_purchased,
			is_trial, trial_ends_at,
			has_subscription, subscription_ends_at,
			status, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	var trialEndsAt, subscriptionEndsAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.CoinsBalance,
		&user.TotalCoinsPurchased,
		&user.IsTrial,
		&trialEndsAt,
		&user.HasSubscription,
		&subscriptionEndsAt,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		log.WithError(err).WithField("email", email).Error("Failed to get user by email")
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if trialEndsAt.Valid {
		user.TrialEndsAt = &trialEndsAt.Time
	}
	if subscriptionEndsAt.Valid {
		user.SubscriptionEndsAt = &subscriptionEndsAt.Time
	}

	return &user, nil
}

func (r *PostgresUserRepository) UpdateEmail(ctx context.Context, userID string, email string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"user_id": userID,
		"email":   email,
	}).Info("Updating user email atomically")

	query := `UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, email, userID)
	if err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to update user email")
		return fmt.Errorf("failed to update email: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not determine rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	log.WithField("user_id", userID).Info("User email successfully updated")
	return nil
}

func (r *PostgresUserRepository) UpdateName(ctx context.Context, userID string, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"user_id": userID,
		"name":    name,
	}).Info("Updating user name atomically")

	query := `UPDATE users SET name = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, name, userID)
	if err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to update user name")
		return fmt.Errorf("failed to update name: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not determine rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	log.WithField("user_id", userID).Info("User name successfully updated")
	return nil
}

func (r *PostgresUserRepository) UpdateStatus(ctx context.Context, userID string, status string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"user_id": userID,
		"status":  status,
	}).Info("Updating user status atomically")

	query := `UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, status, userID)
	if err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to update user status")
		return fmt.Errorf("failed to update status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not determine rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	log.WithField("user_id", userID).Info("User status successfully updated")
	return nil
}

func (r *PostgresUserRepository) AddCoinsAtomic(ctx context.Context, userID string, coins int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if coins <= 0 {
		return fmt.Errorf("coins must be greater than 0")
	}

	log.WithFields(log.Fields{
		"user_id": userID,
		"coins":   coins,
	}).Info("Atomically adding coins to user")

	query := `
		UPDATE users SET
			coins_balance = coins_balance + $1,
			total_coins_purchased = total_coins_purchased + $1,
			updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, coins, userID)
	if err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to add coins atomically")
		return fmt.Errorf("failed to add coins: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not determine rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	log.WithField("user_id", userID).Info("Coins successfully added atomically")
	return nil
}

func (r *PostgresUserRepository) DeductCoinsAtomic(ctx context.Context, userID string, coins int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if coins <= 0 {
		return fmt.Errorf("coins must be greater than 0")
	}

	log.WithFields(log.Fields{
		"user_id": userID,
		"coins":   coins,
	}).Info("Atomically deducting coins from user")

	query := `
		UPDATE users SET
			coins_balance = coins_balance - $1,
			updated_at = NOW()
		WHERE id = $2
		  AND coins_balance >= $1
	`

	result, err := r.db.ExecContext(ctx, query, coins, userID)
	if err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to deduct coins atomically")
		return fmt.Errorf("failed to deduct coins: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not determine rows affected: %w", err)
	}

	if rowsAffected == 0 {
		_, err := r.GetByID(ctx, userID)
		if err != nil {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("insufficient coins balance")
	}

	log.WithField("user_id", userID).Info("Coins successfully deducted atomically")
	return nil
}

func (r *PostgresUserRepository) ActivateSubscriptionAtomic(ctx context.Context, userID string, isTrial bool, trialEndsAt *time.Time, subscriptionEndsAt *time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"user_id":              userID,
		"is_trial":             isTrial,
		"subscription_ends_at": subscriptionEndsAt,
	}).Info("Atomically activating subscription")

	query := `
		UPDATE users SET
			is_trial = $1,
			trial_ends_at = $2,
			has_subscription = true,
			subscription_ends_at = $3,
			updated_at = NOW()
		WHERE id = $4
		  AND has_subscription = false
	`

	result, err := r.db.ExecContext(ctx, query, isTrial, trialEndsAt, subscriptionEndsAt, userID)
	if err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to activate subscription atomically")
		return fmt.Errorf("failed to activate subscription: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not determine rows affected: %w", err)
	}

	if rowsAffected == 0 {
		_, err := r.GetByID(ctx, userID)
		if err != nil {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("subscription already active")
	}

	log.WithField("user_id", userID).Info("Subscription successfully activated atomically")
	return nil
}

func (r *PostgresUserRepository) RenewSubscriptionAtomic(ctx context.Context, userID string, subscriptionEndsAt *time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"user_id":              userID,
		"subscription_ends_at": subscriptionEndsAt,
	}).Info("Atomically renewing subscription")

	query := `
		UPDATE users SET
			subscription_ends_at = $1,
			updated_at = NOW()
		WHERE id = $2
		  AND has_subscription = true
	`

	result, err := r.db.ExecContext(ctx, query, subscriptionEndsAt, userID)
	if err != nil {
		log.WithError(err).WithField("user_id", userID).Error("Failed to renew subscription atomically")
		return fmt.Errorf("failed to renew subscription: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not determine rows affected: %w", err)
	}

	if rowsAffected == 0 {
		_, err := r.GetByID(ctx, userID)
		if err != nil {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("user does not have an active subscription")
	}

	log.WithField("user_id", userID).Info("Subscription successfully renewed atomically")
	return nil
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithField("user_id", id).Info("Deleting user from database")

	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		log.WithError(err).WithField("user_id", id).Error("Failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not determine rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	log.WithField("user_id", id).Info("User successfully deleted")
	return nil
}

func (r *PostgresUserRepository) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, email, name,
			coins_balance, total_coins_purchased,
			is_trial, trial_ends_at,
			has_subscription, subscription_ends_at,
			status, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		log.WithError(err).Error("Failed to list users")
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		var trialEndsAt, subscriptionEndsAt sql.NullTime

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.CoinsBalance,
			&user.TotalCoinsPurchased,
			&user.IsTrial,
			&trialEndsAt,
			&user.HasSubscription,
			&subscriptionEndsAt,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

		if err != nil {
			log.WithError(err).Error("Failed to scan user row")
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}

		if trialEndsAt.Valid {
			user.TrialEndsAt = &trialEndsAt.Time
		}
		if subscriptionEndsAt.Valid {
			user.SubscriptionEndsAt = &subscriptionEndsAt.Time
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		log.WithError(err).Error("Error iterating over user rows")
		return nil, fmt.Errorf("error iterating over user rows: %w", err)
	}

	return users, nil
}
