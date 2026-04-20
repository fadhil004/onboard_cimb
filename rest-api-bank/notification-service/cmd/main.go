package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"microservices-bank/notification-service/config"
	"microservices-bank/notification-service/handler"
	"microservices-bank/notification-service/pkg/logger"

	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func main() {
	godotenv.Load()

	logger.InitLogger()
	defer logger.Logger.Sync()

	database := config.InitDB()
	config.RunMigrations(database)

	// Wait for Kafka to be reachable before starting consumers
	brokers := kafkaBrokers()
	if err := waitForKafka(brokers[0], 60*time.Second); err != nil {
		log.Fatal("[Kafka] Could not connect to broker:", err)
	}
	log.Printf("[Kafka] Connected to broker: %s", brokers[0])

	// Ensure consumed topics exist
	ensureTopics(brokers, []string{
		"account.creation",
		"account.transaction",
		"account.balance_change",
	})

	callbackHandler := handler.NewCallbackHandler(database)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumers for all topics
	topics := []struct {
		GroupID string
		Topic   string
	}{
		{"notification-account-creation", "account.creation"},
		{"notification-transaction", "account.transaction"},
		{"notification-balance-change", "account.balance_change"},
	}

	for _, t := range topics {
		go startConsumer(ctx, t.GroupID, t.Topic, callbackHandler)
	}

	// Simple HTTP server for health check
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "notification-service",
		})
	})

	// Endpoint to list recent notifications
	mux.HandleFunc("GET /notifications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var logs []map[string]interface{}
		rows, err := database.Query(`
			SELECT id, event_type, event_id, topic, callback_status, created_at 
			FROM notification_logs 
			ORDER BY created_at DESC LIMIT 50
		`)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var id int
			var eventType, eventID, topic, callbackStatus string
			var createdAt time.Time
			rows.Scan(&id, &eventType, &eventID, &topic, &callbackStatus, &createdAt)
			logs = append(logs, map[string]interface{}{
				"id":              id,
				"event_type":      eventType,
				"event_id":        eventID,
				"topic":           topic,
				"callback_status": callbackStatus,
				"created_at":      createdAt,
			})
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"responseCode": "200",
			"responseDesc": "Success",
			"data":         logs,
		})
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", httpPort),
		Handler: mux,
	}

	go func() {
		log.Printf("[HTTP] Notification Service running on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error:", err)
		}
	}()

	log.Println("[Notification Service] Running — consuming events from Kafka")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)

	log.Println("Notification Service exited gracefully")
}

func kafkaBrokers() []string {
	raw := os.Getenv("KAFKA_BROKERS")
	if raw == "" {
		return []string{"kafka:29092"}
	}
	return strings.Split(raw, ",")
}

func waitForKafka(broker string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", broker, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		log.Printf("[Kafka] Waiting for broker %s...", broker)
		time.Sleep(3 * time.Second)
	}
	return context.DeadlineExceeded
}

func ensureTopics(brokers []string, topics []string) {
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		log.Println("[Kafka] ensureTopics: dial failed:", err)
		return
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		log.Println("[Kafka] ensureTopics: get controller failed:", err)
		return
	}

	ctrlConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		log.Println("[Kafka] ensureTopics: dial controller failed:", err)
		return
	}
	defer ctrlConn.Close()

	var configs []kafka.TopicConfig
	for _, t := range topics {
		configs = append(configs, kafka.TopicConfig{
			Topic:             t,
			NumPartitions:     3,
			ReplicationFactor: 1,
		})
	}

	if err := ctrlConn.CreateTopics(configs...); err != nil {
		log.Println("[Kafka] ensureTopics (non-fatal):", err)
		return
	}
	log.Printf("[Kafka] Topics ensured: %v", topics)
}

func startConsumer(ctx context.Context, groupID, topic string, h *handler.CallbackHandler) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        kafkaBrokers(),
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
		MaxWait:        500 * time.Millisecond,
		ReadBackoffMin: 100 * time.Millisecond,
		ReadBackoffMax: 1 * time.Second,
		Logger:         kafka.LoggerFunc(func(msg string, args ...interface{}) {}),
		ErrorLogger:    kafka.LoggerFunc(log.Printf),
	})
	defer reader.Close()

	log.Printf("[Kafka] Consumer started: %s (group: %s)", topic, groupID)

	backoff := 500 * time.Millisecond
	const maxBackoff = 30 * time.Second

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Logger.Error("kafka fetch error", zap.String("topic", topic), zap.Error(err))
			// Exponential backoff to handle transient errors like Group Load In Progress
			// and connection issues without hammering the broker
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}
		// Reset backoff on successful fetch
		backoff = 500 * time.Millisecond

		if err := h.HandleEvent(ctx, msg.Topic, string(msg.Key), msg.Value); err != nil {
			logger.Logger.Error("handler error",
				zap.String("topic", msg.Topic),
				zap.String("key", string(msg.Key)),
				zap.Error(err),
			)
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			logger.Logger.Warn("kafka commit error", zap.String("topic", msg.Topic), zap.Error(err))
		}
	}
}
