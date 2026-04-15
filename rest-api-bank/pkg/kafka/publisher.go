package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"rest-api-bank/helper"
	"rest-api-bank/pkg/logger"
)

// Publisher adalah interface yang di-inject ke service.
// Dengan interface ini, service tidak perlu tau detail Kafka —
// bisa di-mock saat testing tanpa butuh broker beneran.
type Publisher interface {
	AccountCreated(ctx context.Context, evt AccountCreatedEvent)
	TransactionCreated(ctx context.Context, evt TransactionCreatedEvent)
	BalanceChanged(ctx context.Context, evt BalanceChangedEvent)
}

// ── KafkaPublisher — implementasi real ───────────────────────────

type KafkaPublisher struct {
	writer  *kafka.Writer
	timeout time.Duration
}

// NewKafkaPublisher membuat publisher baru dari writer yang sudah diinit.
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

func (p *KafkaPublisher) TransactionCreated(ctx context.Context, evt TransactionCreatedEvent) {
	evt.EventID = uuid.NewString()
	evt.EventType = "account.transaction.created"
	evt.OccurredAt = time.Now()
	p.publishAsync(ctx, TopicTransaction, evt.SourceAccountNo, evt)
}

func (p *KafkaPublisher) BalanceChanged(ctx context.Context, evt BalanceChangedEvent) {
	evt.EventID = uuid.NewString()
	evt.EventType = "account.balance_change"
	evt.OccurredAt = time.Now()
	p.publishAsync(ctx, TopicBalanceChange, evt.AccountNumber, evt)
}

// publishAsync mengirim pesan di goroutine terpisah dengan context dan timeout sendiri.
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
			// kirim ke dead letter queue untuk retry
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

// sendDeadLetter mencatat pesan yang gagal ke topic dead letter
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

// ── NoopPublisher — untuk testing / mode tanpa Kafka ─────────────

// NoopPublisher tidak melakukan apa-apa. Dipakai di:
// - unit test (inject ini, tidak perlu broker)
// - local dev tanpa Kafka
// type NoopPublisher struct{}

// func (n *NoopPublisher) AccountCreated(_ context.Context, evt AccountCreatedEvent) {
// 	logger.Logger.Debug("noop: AccountCreated",
// 		zap.String("account_number", evt.AccountNumber),
// 	)
// }

// func (n *NoopPublisher) TransactionCreated(_ context.Context, evt TransactionCreatedEvent) {
// 	logger.Logger.Debug("noop: TransactionCreated",
// 		zap.String("transaction_id", evt.TransactionID),
// 	)
// }

// func (n *NoopPublisher) BalanceChanged(_ context.Context, evt BalanceChangedEvent) {
// 	logger.Logger.Debug("noop: BalanceChanged",
// 		zap.String("account_number", evt.AccountNumber),
// 		zap.String("event_type", evt.EventType),
// 	)
// }

// func traceIDFromCtx(ctx context.Context) string {
// 	if v := ctx.Value("trace_id"); v != nil {
// 		if s, ok := v.(string); ok {
// 			return s
// 		}
// 	}
// 	return ""
// }
