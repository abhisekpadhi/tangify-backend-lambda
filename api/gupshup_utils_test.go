package main

import (
	"encoding/json"
	"testing"
)

func TestGupshupTemplateFormMatchesDashboardCurl(t *testing.T) {
	t.Parallel()

	form, err := gupshupTemplateForm(
		"917855074030",
		"919439831236",
		"HouseOfOdia",
		"a8085178-7d66-4223-826d-25d89aa315d0",
		[]string{"12", "112"},
	)
	if err != nil {
		t.Fatal(err)
	}

	if got := form.Get("channel"); got != "whatsapp" {
		t.Fatalf("channel: got %q", got)
	}
	if got := form.Get("source"); got != "917855074030" {
		t.Fatalf("source: got %q", got)
	}
	if got := form.Get("destination"); got != "919439831236" {
		t.Fatalf("destination: got %q", got)
	}
	if got := form.Get("src.name"); got != "HouseOfOdia" {
		t.Fatalf("src.name: got %q", got)
	}

	var template struct {
		ID     string   `json:"id"`
		Params []string `json:"params"`
	}
	if err := json.Unmarshal([]byte(form.Get("template")), &template); err != nil {
		t.Fatalf("template json: %v", err)
	}
	if template.ID != "a8085178-7d66-4223-826d-25d89aa315d0" {
		t.Fatalf("template id: got %q", template.ID)
	}
	if len(template.Params) != 2 || template.Params[0] != "12" || template.Params[1] != "112" {
		t.Fatalf("template params: got %#v", template.Params)
	}
}

func TestGupshupOTPTemplateHasOneParam(t *testing.T) {
	t.Parallel()

	form, err := gupshupTemplateForm(
		"917855074030",
		"919439831236",
		"HouseOfOdia",
		gupshupWhatsAppTemplates.OTP.DefaultID,
		[]string{"4821"},
	)
	if err != nil {
		t.Fatal(err)
	}
	var template struct {
		ID     string   `json:"id"`
		Params []string `json:"params"`
	}
	if err := json.Unmarshal([]byte(form.Get("template")), &template); err != nil {
		t.Fatalf("template json: %v", err)
	}
	if template.ID != gupshupWhatsAppTemplates.OTP.DefaultID {
		t.Fatalf("template id: got %q", template.ID)
	}
	if len(template.Params) != 1 || template.Params[0] != "4821" {
		t.Fatalf("template params: got %#v", template.Params)
	}
}

func TestGupshupTemplateIDPrefersEnv(t *testing.T) {
	t.Setenv(gupshupWhatsAppTemplates.OTP.EnvKey, "override-otp-id")
	if got := gupshupWhatsAppTemplates.OTP.ID(); got != "override-otp-id" {
		t.Fatalf("ID() env override: got %q", got)
	}
	t.Setenv(gupshupWhatsAppTemplates.OTP.EnvKey, "")
	if got := gupshupWhatsAppTemplates.OTP.ID(); got != gupshupWhatsAppTemplates.OTP.DefaultID {
		t.Fatalf("ID() default: got %q", got)
	}
}

func TestGupshupAccountFromEnv(t *testing.T) {
	t.Setenv("GUPSHUP_API_KEY", "sk_test")
	t.Setenv("GUPSHUP_SOURCE", "917855074030")
	t.Setenv("GUPSHUP_APP_NAME", "HouseOfOdia")
	account, err := gupshupAccountFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if account.APIKey != "sk_test" || account.Source != "917855074030" || account.AppName != "HouseOfOdia" {
		t.Fatalf("account: %+v", account)
	}
	t.Setenv("GUPSHUP_API_KEY", "")
	if _, err := gupshupAccountFromEnv(); err == nil {
		t.Fatal("expected missing API key error")
	}
}
