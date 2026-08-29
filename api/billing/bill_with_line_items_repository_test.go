package billing

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestWalletPointsUpdateExpressionEarnOnly(t *testing.T) {
	t.Parallel()
	got := walletPointsUpdateExpression(0, 12)
	want := "SET points_balance = if_not_exists(points_balance, :zero) + :net, updated_at = :now, lifetime_earned = if_not_exists(lifetime_earned, :zero) + :earn"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestWalletPointsUpdateExpressionRedeemOnly(t *testing.T) {
	t.Parallel()
	got := walletPointsUpdateExpression(5, 0)
	want := "SET points_balance = if_not_exists(points_balance, :zero) + :net, updated_at = :now, lifetime_redeemed = if_not_exists(lifetime_redeemed, :zero) + :redeem"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestWalletPointsUpdateExpressionRedeemAndEarn(t *testing.T) {
	t.Parallel()
	got := walletPointsUpdateExpression(5, 12)
	want := "SET points_balance = if_not_exists(points_balance, :zero) + :net, updated_at = :now, lifetime_redeemed = if_not_exists(lifetime_redeemed, :zero) + :redeem, lifetime_earned = if_not_exists(lifetime_earned, :zero) + :earn"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestWalletPointsUpdateValuesNetDelta(t *testing.T) {
	t.Parallel()
	_, values := walletPointsUpdateValues(5, 12, "user-1", 123)
	net, ok := values[":net"].(*types.AttributeValueMemberN)
	if !ok || net.Value != "7" {
		t.Fatalf(":net=%v want 7", values[":net"])
	}
	_, values = walletPointsUpdateValues(5, 0, "user-1", 123)
	net, ok = values[":net"].(*types.AttributeValueMemberN)
	if !ok || net.Value != "-5" {
		t.Fatalf(":net=%v want -5", values[":net"])
	}
}
