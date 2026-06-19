package loyalty

const TableNamePhoneOTP = "tangify_phone_otp"

const (
	otpTTLSeconds       = 5 * 60
	otpResendCooldownMs = 60 * 1000
	otpMaxAttempts      = 5
)

type PhoneOTP struct {
	Phone        string `json:"phone"`
	OTPHash      string `json:"otp_hash"`
	PendingName  string `json:"pending_name,omitempty"`
	Attempts     int64  `json:"attempts"`
	CreatedAt    int64  `json:"created_at"`
	LastSentAt   int64  `json:"last_sent_at"`
	TTL          int64  `json:"ttl"` // epoch seconds for DynamoDB TTL
}

type SendOTPRequest struct {
	Phone string `json:"phone"`
	Name  string `json:"name,omitempty"`
}

type SendOTPResponse struct {
	Sent bool `json:"sent"`
}

type VerifyOTPRequest struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
	Name  string `json:"name,omitempty"`
}

type VerifyOTPResponse struct {
	UserID        string `json:"user_id"`
	PointsBalance int64  `json:"points_balance"`
	Phone         string `json:"phone"`
}
