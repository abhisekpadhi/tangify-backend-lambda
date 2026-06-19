package menu

import "strings"

// ActiveItemNames returns unique ON menu item names in sheet order.
func ActiveItemNames(items []Item) []string {
	names := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Status)) != "on" {
			continue
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
