package consumer

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	kafkapkg "rest-api-bank/pkg/kafka"
	"rest-api-bank/pkg/logger"
)

// Handler adalah fungsi yang dipanggil untuk setiap message.
type Handler func(ctx context.Context, topic string, key string, value []byte) error

// ConsumerGroup membungkus kafka.Reader dengan auto-reconnect dan graceful shutdown.
type ConsumerGroup struct {
	reader  *kafka.Reader
	handler Handler
}

// NewConsumerGroup membuat consumer group baru.
// groupID unik per use-case — Kafka akan track offset per group secara terpisah.
func NewConsumerGroup(groupID string, topic string, handler Handler) *ConsumerGroup {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        kafkaBrokers(),
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
		MaxWait:        500 * time.Millisecond,
		ReadBackoffMin: 100 * time.Millisecond,
		ReadBackoffMax: 1 * time.Second,
		Logger:         kafka.LoggerFunc(func(msg string, args ...interface{}) {}),
		ErrorLogger:    kafka.LoggerFunc(log.Printf),
	})

	return &ConsumerGroup{reader: reader, handler: handler}
}

// Start memulai consume loop. Blocking — selalu jalankan di goroutine.
// Berhenti bersih saat ctx di-cancel (graceful shutdown).
func (c *ConsumerGroup) Start(ctx context.Context) {
	defer func() {
		if err := c.reader.Close(); err != nil {
			logger.Logger.Error("error closing kafka reader", zap.Error(err))
		}
	}()

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // shutdown normal
			}
			logger.Logger.Error("kafka fetch error",
				zap.String("topic", c.reader.Config().Topic),
				zap.Error(err),
			)
			time.Sleep(time.Second)
			continue
		}

		// Propagate trace-id dari Kafka header ke context
		msgCtx := context.Background()
		for _, h := range msg.Headers {
			if h.Key == "trace-id" {
				msgCtx = context.WithValue(msgCtx, "trace_id", string(h.Value))
			}
		}

		if err := c.handler(msgCtx, msg.Topic, string(msg.Key), msg.Value); err != nil {
			logger.Logger.Error("consumer handler error",
				zap.String("topic", msg.Topic),
				zap.String("key", string(msg.Key)),
				zap.Int64("offset", msg.Offset),
				zap.Error(err),
			)
			// Tetap commit — bad message tidak di-retry infinite.
			// Di production: kirim ke TopicDeadLetter sebelum commit.
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			logger.Logger.Warn("kafka commit error",
				zap.String("topic", msg.Topic),
				zap.Error(err),
			)
		}
	}
}

// Built-in log handlers

func LogAccountCreated(ctx context.Context, topic, key string, value []byte) error {
	var evt kafkapkg.AccountCreatedEvent
	if err := json.Unmarshal(value, &evt); err != nil {
		return err
	}
	logger.Logger.Info("[consumer] account.created",
		zap.String("account_number", evt.AccountNumber),
		zap.String("account_holder", evt.AccountHolder),
		zap.String("reference_no", evt.ReferenceNo),
		zap.Time("occurred_at", evt.OccurredAt),
	)
	return nil
}

func LogTransactionCreated(ctx context.Context, topic, key string, value []byte) error {
	var evt kafkapkg.TransactionCreatedEvent
	if err := json.Unmarshal(value, &evt); err != nil {
		return err
	}
	logger.Logger.Info("[consumer] account.transaction.created",
		zap.String("transaction_id", evt.TransactionID),
		zap.String("from", evt.SourceAccountNo),
		zap.String("to", evt.BeneficiaryAccountNo),
		zap.String("amount", evt.AmountValue+" "+evt.Currency),
		zap.Time("occurred_at", evt.OccurredAt),
	)
	return nil
}

func LogBalanceChanged(ctx context.Context, topic, key string, value []byte) error {
	var evt kafkapkg.BalanceChangedEvent
	if err := json.Unmarshal(value, &evt); err != nil {
		return err
	}
	logger.Logger.Info("[consumer] account.balance_change",
		zap.String("event_type", evt.EventType),
		zap.String("account_number", evt.AccountNumber),
		zap.Int64("amount_changed", evt.AmountChanged),
		zap.Int64("balance_after", evt.BalanceAfter),
		zap.Time("occurred_at", evt.OccurredAt),
	)
	return nil
}

func kafkaBrokers() []string {
	raw := os.Getenv("KAFKA_BROKERS")
	if raw == "" {
		return []string{"kafka:9092"}
	}
	return strings.Split(raw, ",")
}
