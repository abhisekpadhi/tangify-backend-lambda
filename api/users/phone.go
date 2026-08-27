package users

import (
	"fmt"
	"strings"
	"unicode"
)

// CanonicalPhone returns digits-only 91 + 10-digit national number.
// Accepts 10-digit, 91XXXXXXXXXX, +91XXXXXXXXXX, and 0XXXXXXXXXX.
func CanonicalPhone(s string) (string, error) {
	digits := digitsOnly(s)
	switch {
	case len(digits) == 10 && digits[0] >= '6' && digits[0] <= '9':
		return "91" + digits, nil
	case len(digits) == 11 && digits[0] == '0' && digits[1] >= '6' && digits[1] <= '9':
		return "91" + digits[1:], nil
	case len(digits) == 12 && strings.HasPrefix(digits, "91") && digits[2] >= '6' && digits[2] <= '9':
		return digits, nil
	default:
		return "", fmt.Errorf("invalid phone")
	}
}

// PhoneLookupKeys are stored/query forms that may already exist in Dynamo.
func PhoneLookupKeys(s string) []string {
	canon, err := CanonicalPhone(s)
	raw := NormalizePhone(s)
	keys := make([]string, 0, 6)
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		for _, e := range keys {
			if e == k {
				return
			}
		}
		keys = append(keys, k)
	}
	add(raw)
	if err == nil {
		add(canon)
		add(canon[2:])
		add("+" + canon)
		add("0" + canon[2:])
	}
	return keys
}

// DefaultGuestName is used when staff lookup creates a customer without a name.
func DefaultGuestName(phone string) string {
	digits := digitsOnly(phone)
	if len(digits) > 4 {
		digits = digits[len(digits)-4:]
	}
	if digits == "" {
		return "Guest"
	}
	return "Guest " + digits
}

func digitsOnly(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
