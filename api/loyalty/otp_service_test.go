package loyalty

import "testing"

func TestHashOTP_deterministic(t *testing.T) {
	a := hashOTP("secret", "+919876543210", "1234")
	b := hashOTP("secret", "+919876543210", "1234")
	if a != b {
		t.Fatal("hash not deterministic")
	}
	if a == hashOTP("secret", "+919876543210", "5678") {
		t.Fatal("different otp should hash differently")
	}
}

func TestGenerateOTP_format(t *testing.T) {
	for i := 0; i < 20; i++ {
		otp, err := generateOTP()
		if err != nil {
			t.Fatal(err)
		}
		if len(otp) != 4 {
			t.Fatalf("len=%d otp=%q", len(otp), otp)
		}
	}
}

func TestResolveCustomerName(t *testing.T) {
	if got := resolveCustomerName("Alice", "Bob", "+911"); got != "Alice" {
		t.Fatalf("got %q", got)
	}
	if got := resolveCustomerName("", "Bob", "+911"); got != "Bob" {
		t.Fatalf("got %q", got)
	}
	if got := resolveCustomerName("", "", "+919876543210"); got != "Guest 3210" {
		t.Fatalf("got %q", got)
	}
}

func TestOTPErrorStatus(t *testing.T) {
	if OTPErrorStatus(errOTPInvalid) != 401 {
		t.Fatal("invalid should be 401")
	}
	if OTPErrorStatus(errOTPResendTooSoon) != 429 {
		t.Fatal("resend should be 429")
	}
	if OTPErrorStatus(errPhoneRequired) != 400 {
		t.Fatal("phone required should be 400")
	}
}
