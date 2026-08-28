package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	gupshupMessagesURL = "https://api.gupshup.io/wa/api/v1/msg"
	gupshupTemplateURL = "https://api.gupshup.io/wa/api/v1/template/msg"
)

func sendGupshupPlaceholderMessage(ctx context.Context, phone string, name string) error {
	return sendGupshupTextMessage(ctx, phone, fmt.Sprintf("Hi %s, welcome to House of Odia loyalty program. (Template placeholder)", name))
}

func sendGupshupOTPMessage(ctx context.Context, phone, otp string) error {
	return sendGupshupTemplateMessage(ctx, phone, gupshupWhatsAppTemplates.OTP.ID(), []string{otp})
}

func sendGupshupTextMessage(ctx context.Context, phone, text string) error {
	account, err := gupshupAccountFromEnv()
	if err != nil {
		return err
	}

	payload := map[string]any{
		"channel":     "whatsapp",
		"source":      account.Source,
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
	req.Header.Set("apikey", account.APIKey)

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

type gupshupLoyaltyNotifier struct{}

func (gupshupLoyaltyNotifier) NotifyWalletSummary(ctx context.Context, phone string, redeem, earn, balance int64) {
	if strings.TrimSpace(phone) == "" {
		return
	}
	var parts []string
	if redeem > 0 {
		parts = append(parts, fmt.Sprintf("Used %d points", redeem))
	}
	if earn > 0 {
		parts = append(parts, fmt.Sprintf("earned %d points", earn))
	}
	if len(parts) == 0 {
		return
	}
	msg := fmt.Sprintf("House of Odia: %s. Balance: %d points.", strings.Join(parts, ", "), balance)
	if err := sendGupshupTextMessage(ctx, phone, msg); err != nil {
		fmt.Println("gupshup session text:", err)
	}
}

func gupshupTemplateForm(source, destination, appName, templateID string, params []string) (url.Values, error) {
	if params == nil {
		params = []string{}
	}
	templateJSON, err := json.Marshal(map[string]any{
		"id":     templateID,
		"params": params,
	})
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("channel", "whatsapp")
	form.Set("source", digitsOnly(source))
	form.Set("destination", digitsOnly(destination))
	form.Set("src.name", strings.TrimSpace(appName))
	form.Set("template", string(templateJSON))
	return form, nil
}

func sendGupshupTemplateMessage(ctx context.Context, destination, templateID string, params []string) error {
	account, err := gupshupAccountFromEnv()
	if err != nil {
		return err
	}
	if strings.TrimSpace(account.AppName) == "" {
		return fmt.Errorf("GUPSHUP_APP_NAME is required")
	}
	dest := digitsOnly(destination)
	if dest == "" {
		return fmt.Errorf("destination phone required")
	}
	form, err := gupshupTemplateForm(account.Source, dest, account.AppName, templateID, params)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gupshupTemplateURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("apikey", account.APIKey)

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gupshup template status=%d body=%s", resp.StatusCode, string(body))
	}
	return nil
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
