package reviews

import (
	"strings"
	"testing"
)

func TestBuildPromptIncludesRating(t *testing.T) {
	prompt := buildPrompt(4, []string{"Dal Tadka", "Chicken Pakoda"})
	for _, part := range []string{"4/5", "Tangify", "Odia", "Dal Tadka", "Chicken Pakoda"} {
		if !strings.Contains(prompt, part) {
			t.Fatalf("prompt missing %q: %s", part, prompt)
		}
	}
}

func TestBuildPromptWithoutMenuNames(t *testing.T) {
	prompt := buildPrompt(5, nil)
	if !strings.Contains(prompt, "Do not mention any specific dish names") {
		t.Fatalf("expected generic food guidance: %s", prompt)
	}
}

func TestGenerateRejectsInvalidRating(t *testing.T) {
	s := NewService("test-key")
	_, err := s.Generate(t.Context(), GenerateReviewRequest{Rating: 0}, nil)
	if err != errRatingOutOfRange {
		t.Fatalf("expected errRatingOutOfRange, got %v", err)
	}
}
