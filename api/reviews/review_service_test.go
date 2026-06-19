package reviews

import (
	"strings"
	"testing"
)

func TestBuildPromptIncludesRating(t *testing.T) {
	prompt := buildPrompt(4)
	for _, part := range []string{"4/5", "Tangify", "Odia", "3 different"} {
		if !strings.Contains(prompt, part) {
			t.Fatalf("prompt missing %q: %s", part, prompt)
		}
	}
}

func TestGenerateRejectsInvalidRating(t *testing.T) {
	s := NewService("test-key")
	_, err := s.Generate(t.Context(), GenerateReviewRequest{Rating: 0})
	if err != errRatingOutOfRange {
		t.Fatalf("expected errRatingOutOfRange, got %v", err)
	}
}

func TestParseReviewsFromContent(t *testing.T) {
	reviews, err := parseReviewsFromContent(`["first review", "second review", "third review"]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 3 {
		t.Fatalf("expected 3 reviews, got %d", len(reviews))
	}
}

func TestParseReviewsFromContentDedupes(t *testing.T) {
	_, err := parseReviewsFromContent(`["same", "same", "same"]`)
	if err == nil {
		t.Fatal("expected error for duplicate reviews")
	}
}
