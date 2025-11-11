package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"user-service/internal/domain"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	log "github.com/sirupsen/logrus"
)

type AuditPublisher struct {
	producer *kafka.Producer
	topic    string
}

func NewAuditPublisher(bootstrapServers, topic string) (*AuditPublisher, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": bootstrapServers})
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	log.Info("Audit Kafka producer created successfully for user-service")

	return &AuditPublisher{producer: p, topic: topic}, nil
}

func (p *AuditPublisher) Publish(ctx context.Context, event domain.AuditEvent) error {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	deliveryChan := make(chan kafka.Event, 1)
	defer close(deliveryChan)

	if err := p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.topic, Partition: kafka.PartitionAny},
		Key:            []byte(event.EntityID),
		Value:          payload,
		Opaque:         deliveryChan,
	}, nil); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	select {
	case e := <-deliveryChan:
		msg, ok := e.(*kafka.Message)
		if !ok {
			return fmt.Errorf("unexpected event type: %T", e)
		}
		if msg.TopicPartition.Error != nil {
			return fmt.Errorf("delivery failed: %w", msg.TopicPartition.Error)
		}
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("delivery timeout")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *AuditPublisher) Close() {
	log.Info("Closing audit Kafka producer for user-service...")
	p.producer.Flush(15 * 1000)
	p.producer.Close()
}
