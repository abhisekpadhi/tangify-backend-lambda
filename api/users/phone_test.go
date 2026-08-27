package users

import "testing"

func TestCanonicalPhone(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"9876543210", "919876543210"},
		{"91 98765 43210", "919876543210"},
		{"+91 9876543210", "919876543210"},
		{"09876543210", "919876543210"},
	}
	for _, tc := range cases {
		got, err := CanonicalPhone(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
	if _, err := CanonicalPhone("123"); err == nil {
		t.Fatal("expected invalid phone")
	}
}

func TestPhoneLookupKeysIncludesLegacyForms(t *testing.T) {
	t.Parallel()
	keys := PhoneLookupKeys("9876543210")
	want := map[string]bool{
		"9876543210":    true,
		"919876543210":  true,
		"+919876543210": true,
		"09876543210":   true,
	}
	for _, k := range keys {
		delete(want, k)
	}
	if len(want) != 0 {
		t.Fatalf("missing keys %v from %v", want, keys)
	}
}
