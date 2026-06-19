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
	openRouterModel = "openrouter/free"
	reviewCount     = 3
)

var (
	errRatingOutOfRange = errors.New("rating must be between 1 and 5")
	errMissingAPIKey    = errors.New("LLM API key not configured")
	errEmptyReview      = errors.New("LLM returned empty reviews")
)

type openRouterRequest struct {
	Model    string              `json:"model"`
	Messages []openRouterMessage `json:"messages"`
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
		client: &http.Client{Timeout: 35 * time.Second},
	}
}

func (s *Service) Generate(ctx context.Context, req GenerateReviewRequest) (*GenerateReviewResponse, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, errRatingOutOfRange
	}
	if s.apiKey == "" {
		return nil, errMissingAPIKey
	}

	prompt := buildPrompt(req.Rating)
	body, err := json.Marshal(openRouterRequest{
		Model: openRouterModel,
		Messages: []openRouterMessage{
			{
				Role:    "system",
				Content: "You write casual restaurant reviews that sound like real customers from Odisha. Reply with only valid JSON.",
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

	reviews, err := parseReviewsFromContent(parsed.Choices[0].Message.Content)
	if err != nil {
		return nil, err
	}

	return &GenerateReviewResponse{Reviews: reviews}, nil
}

func parseReviewsFromContent(content string) ([]string, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var reviews []string
	if err := json.Unmarshal([]byte(content), &reviews); err != nil {
		return nil, fmt.Errorf("failed to parse reviews JSON: %w", err)
	}

	cleaned := make([]string, 0, len(reviews))
	seen := make(map[string]struct{})
	for _, review := range reviews {
		review = strings.TrimSpace(review)
		review = strings.Trim(review, "\"'`")
		if review == "" {
			continue
		}
		key := strings.ToLower(review)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, review)
	}

	if len(cleaned) < reviewCount {
		return nil, fmt.Errorf("%w: got %d unique reviews", errEmptyReview, len(cleaned))
	}

	return cleaned[:reviewCount], nil
}

func buildPrompt(rating int) string {
	tone := map[int]string{
		5: "very happy, loved the food and vibe",
		4: "good experience, mostly happy with food/service",
		3: "okay-ish, mixed feelings, some good some not",
		2: "disappointed, few things went wrong",
		1: "bad experience, unhappy",
	}[rating]

	return fmt.Sprintf(`Write exactly 3 different short Google Maps style reviews for Tangify restaurant (Odia food place in Bhubaneswar area).

Star rating: %d/5
Mood: %s

Rules for each review:
- 1 to 3 short sentences max, under 280 characters if possible
- Sound like a normal person typing on phone, not polished or marketing
- Mix Odia and English naturally (romanized Odia is fine), like "bhala lagila", "jaldi serve hela", "next time bhi aasiba"
- Mention 1-2 specific things casually (food taste, staff, ambience, waiting time, value)
- No hashtags, no emojis, no bullet points, no "I rate X stars"
- Do not mention being an AI
- All 3 reviews must feel different from each other (different tone, different details)

Return ONLY a JSON array of exactly 3 strings. Example format:
["review one text", "review two text", "review three text"]`, rating, tone)
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
