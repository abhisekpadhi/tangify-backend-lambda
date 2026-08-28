package main

import "testing"

func TestParseLoyaltySessionID(t *testing.T) {
	text := "I want to redeem points for tangify order. order: order-session:ord-1735123456789-k7x9abc"
	got := parseLoyaltySessionID(text)
	want := "order-session:ord-1735123456789-k7x9abc"
	if got != want {
		t.Fatalf("parseLoyaltySessionID: got %q want %q", got, want)
	}
}

func TestParseGupshupInboundTextV3(t *testing.T) {
	body := `{
		"object": "whatsapp_business_account",
		"gs_app_id": "bf9ee64c-3d4d-4ac4-8668-732e577007c4",
		"entry": [{
			"id": "216141188246170",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {
						"display_phone_number": "917855074030",
						"phone_number_id": "207437372456043"
					},
					"contacts": [{
						"profile": { "name": "Guest" },
						"wa_id": "919439831236"
					}],
					"messages": [{
						"from": "919439831236",
						"id": "wamid.test",
						"timestamp": "1705574871",
						"type": "text",
						"text": {
							"body": "I want to redeem points for tangify order. order: freeflow:abc-123"
						}
					}]
				}
			}]
		}]
	}`
	sender, text := parseGupshupInboundText(body)
	if sender != "919439831236" {
		t.Fatalf("sender: got %q", sender)
	}
	if parseLoyaltySessionID(text) != "freeflow:abc-123" {
		t.Fatalf("session: got %q", parseLoyaltySessionID(text))
	}
}

func TestParseGupshupInboundTextV3StatusOnly(t *testing.T) {
	body := `{
		"object": "whatsapp_business_account",
		"entry": [{
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"statuses": [{
						"id": "wamid.test",
						"status": "read",
						"recipient_id": "919439831236",
						"timestamp": "1705574869"
					}]
				}
			}]
		}]
	}`
	sender, text := parseGupshupInboundText(body)
	if sender != "" || text != "" {
		t.Fatalf("expected empty parse for status-only event, got sender=%q text=%q", sender, text)
	}
}

func TestParseGupshupInboundTextLegacy(t *testing.T) {
	body := `{"mobile":"919439831236","text":"I want to redeem points for tangify order. order: freeflow:abc-123"}`
	sender, text := parseGupshupInboundText(body)
	if sender != "919439831236" {
		t.Fatalf("sender: got %q", sender)
	}
	if parseLoyaltySessionID(text) != "freeflow:abc-123" {
		t.Fatalf("session: got %q", parseLoyaltySessionID(text))
	}
}
