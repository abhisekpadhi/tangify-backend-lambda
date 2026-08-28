package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ably/ably-go/ably"
)

type AblyUtils struct {
	realtime *ably.Realtime
	enabled  bool
}

func NewAblyUtils() (*AblyUtils, error) {
	key := os.Getenv("ABLY_KEY")
	if key == "" {
		return &AblyUtils{enabled: false}, nil
	}

	client, err := ably.NewRealtime(ably.WithKey(key))
	if err != nil {
		return nil, err
	}
	return &AblyUtils{realtime: client, enabled: true}, nil
}

func (a *AblyUtils) PublishJSON(ctx context.Context, channelName string, eventName string, payload any) error {
	if a == nil || !a.enabled {
		return nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ch := a.realtime.Channels.Get(channelName)
	return ch.Publish(ctx, eventName, string(b))
}

func kitchenChannel(venueID string) string {
	return fmt.Sprintf("kitchen:%s", venueID)
}

func waiterChannel(venueID string) string {
	return fmt.Sprintf("waiter:%s", venueID)
}

func orderOpsChannel() string {
	if ch := strings.TrimSpace(os.Getenv("ABLY_CHANNEL")); ch != "" {
		return ch
	}
	return "order_ops"
}
