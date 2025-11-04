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
	Update(ctx context.Context, user *domain.User) error
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
			return nil, fmt.Errorf("user not found: %w", err)
		}
		log.WithError(err).WithField("user_id", id).Error("Failed to get user by ID")
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	// Преобразуем sql.NullTime в *time.Time
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
			return nil, fmt.Errorf("user not found: %w", err)
		}
		log.WithError(err).WithField("email", email).Error("Failed to get user by email")
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Преобразуем sql.NullTime в *time.Time
	if trialEndsAt.Valid {
		user.TrialEndsAt = &trialEndsAt.Time
	}
	if subscriptionEndsAt.Valid {
		user.SubscriptionEndsAt = &subscriptionEndsAt.Time
	}

	return &user, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("Updating user in database")

	query := `
		UPDATE users SET
			email = $1,
			name = $2,
			coins_balance = $3,
			total_coins_purchased = $4,
			is_trial = $5,
			trial_ends_at = $6,
			has_subscription = $7,
			subscription_ends_at = $8,
			status = $9,
			updated_at = NOW()
		WHERE id = $10
	`

	result, err := r.db.ExecContext(ctx, query,
		user.Email,
		user.Name,
		user.CoinsBalance,
		user.TotalCoinsPurchased,
		user.IsTrial,
		user.TrialEndsAt,
		user.HasSubscription,
		user.SubscriptionEndsAt,
		user.Status,
		user.ID,
	)

	if err != nil {
		log.WithError(err).WithField("user_id", user.ID).Error("Failed to update user")
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not determine rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	log.WithField("user_id", user.ID).Info("User successfully updated")
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

		// Преобразуем sql.NullTime в *time.Time
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