package config

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaWriter adalah writer global yang dipakai oleh producer.
// Satu writer bisa kirim ke banyak topic — topic ditentukan per pesan.
var KafkaWriter *kafka.Writer

func InitKafka() {
	brokers := kafkaBrokers()

	KafkaWriter = &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.Hash{}, // partition key = accountNumber → Hash balancer
		RequiredAcks:           kafka.RequireAll,
		MaxAttempts:            5,
		WriteBackoffMin:        100 * time.Millisecond,
		WriteBackoffMax:        1 * time.Second,
		BatchSize:              100,
		BatchTimeout:           10 * time.Millisecond,
		Compression:            kafka.Snappy,
		AllowAutoTopicCreation: false, // topic dibuat manual via kafka-init
		Logger:                 kafka.LoggerFunc(func(msg string, args ...interface{}) {}), // silent
		ErrorLogger:            kafka.LoggerFunc(log.Printf),
	}

	// Cek koneksi awal ke broker
	if err := waitForKafka(brokers[0], 30*time.Second); err != nil {
		log.Fatal("[Kafka] Could not connect to broker:", err)
	}
	log.Printf("[Kafka] Writer connected to brokers: %v", brokers)
}

func CloseKafka() {
	if KafkaWriter != nil {
		if err := KafkaWriter.Close(); err != nil {
			log.Println("[Kafka] Error closing writer:", err)
		}
	}
}

func kafkaBrokers() []string {
	raw := os.Getenv("KAFKA_BROKERS")
	if raw == "" {
		return []string{"kafka:9092"}
	}
	return strings.Split(raw, ",")
}

// waitForKafka melakukan TCP dial ke broker sampai berhasil atau timeout.
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

// EnsureTopics membuat topic jika belum ada.
// Dipanggil saat startup sebagai fallback kalau kafka-init tidak jalan.
func EnsureTopics(topics []kafka.TopicConfig) {
	brokers := kafkaBrokers()
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		log.Println("[Kafka] EnsureTopics: dial failed:", err)
		return
	}
	defer conn.Close()

	// Cari controller broker
	controller, err := conn.Controller()
	if err != nil {
		log.Println("[Kafka] EnsureTopics: get controller failed:", err)
		return
	}

	ctrlConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		log.Println("[Kafka] EnsureTopics: dial controller failed:", err)
		return
	}
	defer ctrlConn.Close()

	if err := ctrlConn.CreateTopics(topics...); err != nil {
		// Error ini biasanya TopicAlreadyExists — aman diabaikan
		log.Println("[Kafka] EnsureTopics (non-fatal):", err)
		return
	}
	log.Println("[Kafka] Topics ensured:", len(topics))
}
