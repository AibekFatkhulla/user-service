package service

import (
	"context"
	"time"

	"user-service/internal/domain"
)

type AuditPublisher interface {
	Publish(ctx context.Context, event domain.AuditEvent) error
}

type AuditService struct {
	publisher AuditPublisher
}

func NewAuditService(publisher AuditPublisher) *AuditService {
	return &AuditService{publisher: publisher}
}

func (s *AuditService) RecordUserCreated(ctx context.Context, user *domain.User) error {
	if s == nil || s.publisher == nil || user == nil {
		return nil
	}

	event := domain.AuditEvent{
		Service:    "user-service",
		EventType:  "user_created",
		EntityID:   user.ID,
		Actor:      user.ID,
		OccurredAt: time.Now().UTC(),
		Payload: map[string]interface{}{
			"email":            user.Email,
			"name":             user.Name,
			"coins_balance":    user.CoinsBalance,
			"is_trial":         user.IsTrial,
			"has_subscription": user.HasSubscription,
			"status":           user.Status,
		},
	}

	if user.TrialEndsAt != nil {
		event.Payload["trial_ends_at"] = user.TrialEndsAt
	}
	if user.SubscriptionEndsAt != nil {
		event.Payload["subscription_ends_at"] = user.SubscriptionEndsAt
	}

	return s.publisher.Publish(ctx, event)
}

func (s *AuditService) RecordUserUpdated(ctx context.Context, userID string, changes map[string]interface{}) error {
	if s == nil || s.publisher == nil || len(changes) == 0 {
		return nil
	}

	event := domain.AuditEvent{
		Service:    "user-service",
		EventType:  "user_updated",
		EntityID:   userID,
		Actor:      userID,
		OccurredAt: time.Now().UTC(),
		Payload: map[string]interface{}{
			"changes": changes,
		},
	}

	return s.publisher.Publish(ctx, event)
}

func (s *AuditService) RecordCoinsAdded(ctx context.Context, userID string, amount int64) error {
	if s == nil || s.publisher == nil {
		return nil
	}

	event := domain.AuditEvent{
		Service:    "user-service",
		EventType:  "user_coins_added",
		EntityID:   userID,
		Actor:      userID,
		OccurredAt: time.Now().UTC(),
		Payload: map[string]interface{}{
			"amount": amount,
		},
	}

	return s.publisher.Publish(ctx, event)
}

func (s *AuditService) RecordCoinsDeducted(ctx context.Context, userID string, amount int64) error {
	if s == nil || s.publisher == nil {
		return nil
	}

	event := domain.AuditEvent{
		Service:    "user-service",
		EventType:  "user_coins_deducted",
		EntityID:   userID,
		Actor:      userID,
		OccurredAt: time.Now().UTC(),
		Payload: map[string]interface{}{
			"amount": amount,
		},
	}

	return s.publisher.Publish(ctx, event)
}

func (s *AuditService) RecordSubscriptionEvent(ctx context.Context, userID, eventType string, duration time.Duration, endsAt time.Time) error {
	if s == nil || s.publisher == nil {
		return nil
	}

	event := domain.AuditEvent{
		Service:    "user-service",
		EventType:  eventType,
		EntityID:   userID,
		Actor:      userID,
		OccurredAt: time.Now().UTC(),
		Payload: map[string]interface{}{
			"duration_hours":       duration.Hours(),
			"subscription_ends_at": endsAt,
		},
	}

	return s.publisher.Publish(ctx, event)
}
