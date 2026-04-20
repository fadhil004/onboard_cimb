package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"microservices-bank/account-service/helper"
	"microservices-bank/account-service/pkg/logger"
)

// Publisher adalah interface yang di-inject ke service.
type Publisher interface {
	AccountCreated(ctx context.Context, evt AccountCreatedEvent)
	BalanceChanged(ctx context.Context, evt BalanceChangedEvent)
}

// KafkaPublisher — implementasi real
type KafkaPublisher struct {
	writer  *kafka.Writer
	timeout time.Duration
}

func NewKafkaPublisher(writer *kafka.Writer) *KafkaPublisher {
	return &KafkaPublisher{
		writer:  writer,
		timeout: 5 * time.Second,
	}
}

func (p *KafkaPublisher) AccountCreated(ctx context.Context, evt AccountCreatedEvent) {
	evt.EventID = uuid.NewString()
	evt.EventType = "account.created"
	evt.OccurredAt = time.Now()
	p.publishAsync(ctx, TopicAccountCreation, evt.AccountNumber, evt)
}

func (p *KafkaPublisher) BalanceChanged(ctx context.Context, evt BalanceChangedEvent) {
	evt.EventID = uuid.NewString()
	evt.EventType = "account.balance_change"
	evt.OccurredAt = time.Now()
	p.publishAsync(ctx, TopicBalanceChange, evt.AccountNumber, evt)
}

func (p *KafkaPublisher) publishAsync(reqCtx context.Context, topic, key string, payload interface{}) {
	traceID := helper.GetTraceID(reqCtx)

	go func() {
		pubCtx, cancel := context.WithTimeout(context.Background(), p.timeout)
		defer cancel()

		if err := p.send(pubCtx, topic, key, traceID, payload); err != nil {
			logger.Logger.Warn("kafka publish failed (non-fatal)",
				zap.String("topic", topic),
				zap.String("key", key),
				zap.String("trace_id", traceID),
				zap.Error(err),
			)
			p.sendDeadLetter(topic, key, traceID, payload, err)
		}
	}()
}

func (p *KafkaPublisher) send(ctx context.Context, topic, key, traceID string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{Key: "content-type", Value: []byte("application/json")},
			{Key: "trace-id", Value: []byte(traceID)},
			{Key: "event-id", Value: []byte(uuid.NewString())},
		},
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	logger.Logger.Info("kafka published",
		zap.String("topic", topic),
		zap.String("key", key),
		zap.String("trace_id", traceID),
	)
	return nil
}

func (p *KafkaPublisher) sendDeadLetter(originalTopic, key, traceID string, payload interface{}, cause error) {
	data, err := json.Marshal(map[string]interface{}{
		"originalTopic": originalTopic,
		"key":           key,
		"payload":       payload,
		"error":         cause.Error(),
		"failedAt":      time.Now(),
	})
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	msg := kafka.Message{
		Topic: TopicDeadLetter,
		Key:   []byte(key),
		Value: data,
		Headers: []kafka.Header{
			{Key: "trace-id", Value: []byte(traceID)},
			{Key: "original-topic", Value: []byte(originalTopic)},
		},
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		logger.Logger.Error("dead letter publish also failed",
			zap.String("original_topic", originalTopic),
			zap.String("key", key),
			zap.Error(err),
		)
	} else {
		logger.Logger.Warn("event sent to dead letter queue",
			zap.String("original_topic", originalTopic),
			zap.String("key", key),
		)
	}
}
