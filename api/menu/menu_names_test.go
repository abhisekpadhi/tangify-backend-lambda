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
