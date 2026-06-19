package menu

import "testing"

func TestActiveItemNames(t *testing.T) {
	items := []Item{
		{Status: "ON", Name: "Dal Tadka"},
		{Status: "off", Name: "Hidden Item"},
		{Status: "ON", Name: "Dal Tadka"},
		{Status: "ON", Name: "  Pakoda  "},
		{Status: "ON", Name: ""},
	}

	names := ActiveItemNames(items)
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d: %v", len(names), names)
	}
	if names[0] != "Dal Tadka" || names[1] != "Pakoda" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestActiveItemNamesInCategories(t *testing.T) {
	items := []Item{
		{Status: "ON", Category: "Mains", Name: "Dal Tadka"},
		{Status: "ON", Category: "Desserts", Name: "Rasgulla"},
		{Status: "ON", Category: "starters", Name: "Pakoda"},
		{Status: "off", Category: "Mains", Name: "Off Item"},
	}

	names := ActiveItemNamesInCategories(items, ReviewContextCategories)
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d: %v", len(names), names)
	}
	if names[0] != "Dal Tadka" || names[1] != "Pakoda" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestShuffleStringsPreservesItems(t *testing.T) {
	input := []string{"a", "b", "c", "d"}
	shuffled := ShuffleStrings(input)

	if len(shuffled) != len(input) {
		t.Fatalf("expected length %d, got %d", len(input), len(shuffled))
	}

	counts := map[string]int{}
	for _, value := range shuffled {
		counts[value]++
	}
	for _, value := range input {
		if counts[value] != 1 {
			t.Fatalf("unexpected shuffle result: %v", shuffled)
		}
	}
}
