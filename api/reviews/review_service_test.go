package reviews

import (
	"strings"
	"testing"
)

func TestBuildPromptIncludesRating(t *testing.T) {
	prompt := buildPrompt(4, []string{"Dal Tadka", "Chicken Pakoda"})
	for _, part := range []string{"4/5", "Tangify", "THIS REVIEW MUST FOLLOW"} {
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

func TestRandomPromptStyleVaries(t *testing.T) {
	seen := map[string]struct{}{}
	dialects := map[string]struct{}{}
	for range 30 {
		style := randomPromptStyle()
		key := style.lengthGuide + "|" + style.openingGuide + "|" + style.endingGuide
		seen[key] = struct{}{}
		dialects[style.dialectGuide] = struct{}{}
	}
	if len(seen) < 3 {
		t.Fatalf("expected varied prompt styles, got %d unique combinations", len(seen))
	}
	if len(dialects) < 3 {
		t.Fatalf("expected varied dialect guides, got %d", len(dialects))
	}
}

func TestGenerateRejectsInvalidRating(t *testing.T) {
	s := NewService("test-key")
	_, err := s.Generate(t.Context(), GenerateReviewRequest{Rating: 0}, nil)
	if err != errRatingOutOfRange {
		t.Fatalf("expected errRatingOutOfRange, got %v", err)
	}
}
