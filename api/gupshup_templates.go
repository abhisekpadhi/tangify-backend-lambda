package main

import (
	"fmt"
	"os"
	"strings"
)

// WhatsApp templates used by this Lambda. Change DefaultID here when Gupshup
// issues a new id. Runtime override: set EnvKey on the function (see .env.lambda).
var gupshupWhatsAppTemplates = struct {
	RewardPoint gupshupTemplate
	PointsUsed  gupshupTemplate
	OTP         gupshupTemplate
}{
	RewardPoint: gupshupTemplate{
		Name:      "reward_point",
		EnvKey:    "GUPSHUP_REWARD_POINT_TEMPLATE_ID",
		DefaultID: "a8085178-7d66-4223-826d-25d89aa315d0",
	},
	PointsUsed: gupshupTemplate{
		Name:      "points_used",
		EnvKey:    "GUPSHUP_POINTS_USED_TEMPLATE_ID",
		DefaultID: "47bf6ed2-2d4f-4b35-976f-34874eaa5468",
	},
	OTP: gupshupTemplate{
		Name:      "points_at_counter",
		EnvKey:    "GUPSHUP_OTP_TEMPLATE_ID",
		DefaultID: "4d54f495-4f30-41ed-85ae-c68faed15a78",
	},
}

type gupshupTemplate struct {
	Name      string
	EnvKey    string
	DefaultID string
}

func (t gupshupTemplate) ID() string {
	if v := strings.TrimSpace(os.Getenv(t.EnvKey)); v != "" {
		return v
	}
	return t.DefaultID
}

type gupshupAccount struct {
	APIKey  string
	Source  string
	AppName string
}

func gupshupAccountFromEnv() (gupshupAccount, error) {
	a := gupshupAccount{
		APIKey:  strings.TrimSpace(os.Getenv("GUPSHUP_API_KEY")),
		Source:  strings.TrimSpace(os.Getenv("GUPSHUP_SOURCE")),
		AppName: strings.TrimSpace(os.Getenv("GUPSHUP_APP_NAME")),
	}
	if a.APIKey == "" || a.Source == "" {
		return a, fmt.Errorf("GUPSHUP_API_KEY and GUPSHUP_SOURCE are required")
	}
	return a, nil
}
