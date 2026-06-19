package loyalty

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"tangify-backend-lambda/users"
)

var (
	errOTPNotFound      = errors.New("otp not found or expired")
	errOTPExpired       = errors.New("otp expired")
	errOTPTooManyTries  = errors.New("too many invalid attempts")
	errOTPInvalid       = errors.New("invalid otp")
	errOTPResendTooSoon = errors.New("please wait before requesting another otp")
	errPhoneRequired    = errors.New("phone required")
	errOTPRequired      = errors.New("otp required")
)

type CustomerCreator interface {
	CreateOrGetCustomer(ctx context.Context, phone, name string, now int64) (*users.UserPublic, error)
}

type OTPService struct {
	otpRepo    *OTPRepository
	walletRepo *Repository
	users      CustomerCreator
	secret     string
	sendOTP    func(ctx context.Context, phone, otp string) error
}

func NewOTPService(
	otpRepo *OTPRepository,
	walletRepo *Repository,
	users CustomerCreator,
	secret string,
	sendOTP func(ctx context.Context, phone, otp string) error,
) *OTPService {
	return &OTPService{
		otpRepo:    otpRepo,
		walletRepo: walletRepo,
		users:      users,
		secret:     secret,
		sendOTP:    sendOTP,
	}
}

func (s *OTPService) Send(ctx context.Context, req SendOTPRequest, nowMs int64) (*SendOTPResponse, error) {
	phone := users.NormalizePhone(req.Phone)
	if phone == "" {
		return nil, errPhoneRequired
	}

	existing, err := s.otpRepo.Get(ctx, phone)
	if err != nil {
		return nil, err
	}
	if existing != nil && nowMs-existing.LastSentAt < otpResendCooldownMs {
		return nil, errOTPResendTooSoon
	}

	otp, err := generateOTP()
	if err != nil {
		return nil, err
	}

	nowSec := nowMs / 1000
	row := &PhoneOTP{
		Phone:       phone,
		OTPHash:     hashOTP(s.secret, phone, otp),
		PendingName: strings.TrimSpace(req.Name),
		Attempts:    0,
		CreatedAt:   nowMs,
		LastSentAt:  nowMs,
		TTL:         nowSec + otpTTLSeconds,
	}
	if err := s.otpRepo.Put(ctx, row); err != nil {
		return nil, err
	}
	if err := s.sendOTP(ctx, phone, otp); err != nil {
		return nil, err
	}
	return &SendOTPResponse{Sent: true}, nil
}

func (s *OTPService) Verify(ctx context.Context, req VerifyOTPRequest, nowMs int64) (*VerifyOTPResponse, error) {
	phone := users.NormalizePhone(req.Phone)
	otp := strings.TrimSpace(req.OTP)
	if phone == "" {
		return nil, errPhoneRequired
	}
	if otp == "" {
		return nil, errOTPRequired
	}

	row, err := s.otpRepo.Get(ctx, phone)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errOTPNotFound
	}
	nowSec := nowMs / 1000
	if row.TTL <= nowSec {
		return nil, errOTPExpired
	}
	if row.Attempts >= otpMaxAttempts {
		return nil, errOTPTooManyTries
	}

	expected := hashOTP(s.secret, phone, otp)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(row.OTPHash)) != 1 {
		_ = s.otpRepo.IncrementAttempts(ctx, phone)
		return nil, errOTPInvalid
	}

	if err := s.otpRepo.Delete(ctx, phone); err != nil {
		return nil, err
	}

	name := resolveCustomerName(req.Name, row.PendingName, phone)
	user, err := s.users.CreateOrGetCustomer(ctx, phone, name, nowMs)
	if err != nil {
		return nil, err
	}

	w, err := s.walletRepo.GetWallet(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if w.UpdatedAt == 0 && w.PointsBalance == 0 && w.LifetimeEarned == 0 && w.LifetimeRedeemed == 0 {
		w.UserID = user.ID
		w.UpdatedAt = nowMs
		if err := s.walletRepo.PutWallet(ctx, w); err != nil {
			return nil, err
		}
	}

	return &VerifyOTPResponse{
		UserID:        user.ID,
		PointsBalance: w.PointsBalance,
		Phone:         phone,
	}, nil
}

func resolveCustomerName(verifyName, pendingName, phone string) string {
	if n := strings.TrimSpace(verifyName); n != "" {
		return n
	}
	if n := strings.TrimSpace(pendingName); n != "" {
		return n
	}
	suffix := phone
	if len(suffix) > 4 {
		suffix = suffix[len(suffix)-4:]
	}
	return "Guest " + suffix
}

func generateOTP() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	n := (int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])) % 10000
	if n < 0 {
		n = -n
	}
	return fmt.Sprintf("%04d", n), nil
}

func hashOTP(secret, phone, otp string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(phone + ":" + otp))
	return hex.EncodeToString(mac.Sum(nil))
}

// sendOTPWhatsApp is set from main package to avoid import cycle.
var sendOTPWhatsApp = func(ctx context.Context, phone, otp string) error {
	return fmt.Errorf("sendOTPWhatsApp not configured")
}

// OTPErrorStatus maps OTP errors to HTTP status codes.
func OTPErrorStatus(err error) int {
	switch {
	case errors.Is(err, errOTPInvalid), errors.Is(err, errOTPNotFound), errors.Is(err, errOTPExpired), errors.Is(err, errOTPTooManyTries):
		return 401
	case errors.Is(err, errOTPResendTooSoon):
		return 429
	default:
		return 400
	}
}
