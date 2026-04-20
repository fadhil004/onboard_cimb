package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"microservices-bank/notification-service/pkg/logger"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// CallbackHandler consumes Kafka events, logs them, and sends HTTP callbacks to the partner.
type CallbackHandler struct {
	DB          *sqlx.DB
	CallbackURL string
	HTTPClient  *http.Client
}

func NewCallbackHandler(db *sqlx.DB) *CallbackHandler {
	callbackURL := os.Getenv("CALLBACK_URL")

	return &CallbackHandler{
		DB:          db,
		CallbackURL: callbackURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// HandleEvent processes a Kafka event: stores in DB and sends HTTP callback.
func (h *CallbackHandler) HandleEvent(ctx context.Context, topic string, key string, value []byte) error {
	// Parse payload
	var payload map[string]interface{}
	if err := json.Unmarshal(value, &payload); err != nil {
		logger.Logger.Error("[notification] failed to parse event payload",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return err
	}

	eventType, _ := payload["eventType"].(string)
	eventID, _ := payload["eventId"].(string)

	logger.Logger.Info("[notification] received event",
		zap.String("topic", topic),
		zap.String("event_type", eventType),
		zap.String("event_id", eventID),
		zap.String("key", key),
	)

	// Store notification log
	payloadJSON, _ := json.Marshal(payload)
	_, err := h.DB.ExecContext(ctx, `
		INSERT INTO notification_logs (event_type, event_id, topic, payload, callback_url, callback_status)
		VALUES ($1, $2, $3, $4, $5, 'PENDING')
	`, eventType, eventID, topic, payloadJSON, h.CallbackURL)
	if err != nil {
		logger.Logger.Error("[notification] failed to store notification log", zap.Error(err))
	}

	// Build callback payload
	callbackPayload := map[string]interface{}{
		"eventType":  eventType,
		"eventId":    eventID,
		"topic":      topic,
		"data":       payload,
		"notifiedAt": time.Now().Format(time.RFC3339),
	}

	// If this is a transaction event, include transaction status
	if topic == "account.transaction" {
		callbackPayload["transactionStatus"] = "SUCCESS"
		if status, ok := payload["status"].(string); ok {
			callbackPayload["transactionStatus"] = status
		}
	}

	// Send HTTP callback
	callbackJSON, err := json.Marshal(callbackPayload)
	if err != nil {
		logger.Logger.Error("[notification] failed to marshal callback payload", zap.Error(err))
		return err
	}

	callbackStatus := "SUCCESS"
	callbackResponse := ""

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.CallbackURL, bytes.NewReader(callbackJSON))
	if err != nil {
		logger.Logger.Error("[notification] failed to create callback request", zap.Error(err))
		callbackStatus = "FAILED"
		callbackResponse = err.Error()
	} else {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Event-Type", eventType)
		req.Header.Set("X-Event-ID", eventID)

		resp, err := h.HTTPClient.Do(req)
		if err != nil {
			logger.Logger.Error("[notification] callback request failed",
				zap.String("callback_url", h.CallbackURL),
				zap.Error(err),
			)
			callbackStatus = "FAILED"
			callbackResponse = err.Error()
		} else {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			callbackResponse = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				callbackStatus = "SUCCESS"
				logger.Logger.Info("[notification] callback sent successfully",
					zap.String("callback_url", h.CallbackURL),
					zap.Int("status_code", resp.StatusCode),
					zap.String("event_type", eventType),
				)
			} else {
				callbackStatus = "FAILED"
				logger.Logger.Warn("[notification] callback returned non-2xx",
					zap.String("callback_url", h.CallbackURL),
					zap.Int("status_code", resp.StatusCode),
				)
			}
		}
	}

	// Update notification log with callback result
	h.DB.ExecContext(ctx, `
		UPDATE notification_logs SET callback_status=$1, callback_response=$2
		WHERE event_id=$3
	`, callbackStatus, callbackResponse, eventID)

	return nil
}
