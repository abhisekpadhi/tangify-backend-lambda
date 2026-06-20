package reviews

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	openRouterURL   = "https://openrouter.ai/api/v1/chat/completions"
	openRouterModel = "openai/gpt-oss-120b:free"
)

var (
	errRatingOutOfRange = errors.New("rating must be between 1 and 5")
	errMissingAPIKey    = errors.New("LLM API key not configured")
	errEmptyReview      = errors.New("LLM returned empty review")
)

type openRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []openRouterMessage `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type Service struct {
	apiKey string
	client *http.Client
}

func NewService(apiKey string) *Service {
	return &Service{
		apiKey: strings.TrimSpace(apiKey),
		client: &http.Client{Timeout: 285 * time.Second},
	}
}

func (s *Service) Generate(ctx context.Context, req GenerateReviewRequest, menuItemNames []string) (*GenerateReviewResponse, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, errRatingOutOfRange
	}
	if s.apiKey == "" {
		return nil, errMissingAPIKey
	}

	prompt := buildPrompt(req.Rating, menuItemNames)
	body, err := json.Marshal(openRouterRequest{
		Model:       openRouterModel,
		Temperature: 0.95,
		Messages: []openRouterMessage{
			{
				Role:    "system",
				Content: "You write casual Google Maps reviews for an Odia restaurant in Bhubaneswar. Each review uses a different language mix: plain English, Odia-English, Hindi, or Hinglish — follow the user prompt exactly. Reply with only the review text.",
			},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openrouter status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var parsed openRouterResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("openrouter error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, errEmptyReview
	}

	review := strings.TrimSpace(parsed.Choices[0].Message.Content)
	review = strings.Trim(review, "\"'`")
	if review == "" {
		return nil, errEmptyReview
	}

	return &GenerateReviewResponse{Review: review}, nil
}

func ErrorStatus(err error) int {
	switch {
	case errors.Is(err, errRatingOutOfRange):
		return http.StatusBadRequest
	case errors.Is(err, errMissingAPIKey):
		return http.StatusServiceUnavailable
	default:
		return http.StatusBadGateway
	}
}
