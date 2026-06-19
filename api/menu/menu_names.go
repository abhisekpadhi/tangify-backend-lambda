package menu

import (
	"math/rand/v2"
	"strings"
)

// ReviewContextCategories are menu categories passed into review generation prompts.
var ReviewContextCategories = []string{
	"thali",
	"mains",
	"lassi",
	"starters",
	"combo",
}

// ActiveItemNames returns unique ON menu item names in sheet order.
func ActiveItemNames(items []Item) []string {
	return ActiveItemNamesInCategories(items, nil)
}

// ActiveItemNamesInCategories returns unique ON item names limited to allowed categories.
// Category matching is case-insensitive. When allowedCategories is empty, all ON items are included.
func ActiveItemNamesInCategories(items []Item, allowedCategories []string) []string {
	allowed := categorySet(allowedCategories)
	names := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Status)) != "on" {
			continue
		}
		if len(allowed) > 0 {
			category := strings.ToLower(strings.TrimSpace(item.Category))
			if _, ok := allowed[category]; !ok {
				continue
			}
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, name)
	}
	return names
}

// ShuffleStrings returns a shuffled copy of values.
func ShuffleStrings(values []string) []string {
	if len(values) <= 1 {
		out := make([]string, len(values))
		copy(out, values)
		return out
	}
	out := make([]string, len(values))
	copy(out, values)
	rand.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}

func categorySet(categories []string) map[string]struct{} {
	if len(categories) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		category = strings.ToLower(strings.TrimSpace(category))
		if category == "" {
			continue
		}
		set[category] = struct{}{}
	}
	return set
}
