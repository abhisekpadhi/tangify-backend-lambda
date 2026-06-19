package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const gupshupMessagesURL = "https://api.gupshup.io/wa/api/v1/msg"

func sendGupshupPlaceholderMessage(ctx context.Context, phone string, name string) error {
	return sendGupshupTextMessage(ctx, phone, fmt.Sprintf("Hi %s, welcome to House of Odia loyalty program. (Template placeholder)", name))
}

func sendGupshupOTPMessage(ctx context.Context, phone, otp string) error {
	return sendGupshupTextMessage(ctx, phone, fmt.Sprintf("Your House of Odia OTP is %s. Valid for 5 minutes.", otp))
}

func sendGupshupTextMessage(ctx context.Context, phone, text string) error {
	apiKey := strings.TrimSpace(os.Getenv("GUPSHUP_API_KEY"))
	source := strings.TrimSpace(os.Getenv("GUPSHUP_SOURCE"))
	if apiKey == "" || source == "" {
		return fmt.Errorf("GUPSHUP_API_KEY and GUPSHUP_SOURCE are required")
	}

	payload := map[string]any{
		"channel":     "whatsapp",
		"source":      source,
		"destination": phone,
		"message": map[string]any{
			"type": "text",
			"text": text,
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gupshupMessagesURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apiKey)

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gupshup status=%d body=%s", resp.StatusCode, string(body))
	}
	return nil
}
