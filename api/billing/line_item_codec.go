package billing

import (
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func encodeLineItem(li LineItem) map[string]types.AttributeValue {
	li = NormalizeLineItem(li)
	im := map[string]types.AttributeValue{
		"id":       &types.AttributeValueMemberS{Value: li.ID},
		"name":     &types.AttributeValueMemberS{Value: li.Name},
		"quantity": &types.AttributeValueMemberN{Value: strconv.Itoa(li.Quantity)},
		"price":    &types.AttributeValueMemberN{Value: strconv.FormatInt(li.Price, 10)},
		"status":   &types.AttributeValueMemberS{Value: li.Status},
	}
	if li.MenuItemID != "" {
		im["menu_item_id"] = &types.AttributeValueMemberS{Value: li.MenuItemID}
	}
	if li.Category != "" {
		im["category"] = &types.AttributeValueMemberS{Value: li.Category}
	}
	if li.InternalName != "" {
		im["internal_name"] = &types.AttributeValueMemberS{Value: li.InternalName}
	}
	if li.UserOverride != nil {
		ov := map[string]types.AttributeValue{}
		if li.UserOverride.Quantity != nil {
			ov["quantity"] = &types.AttributeValueMemberN{Value: strconv.Itoa(*li.UserOverride.Quantity)}
		}
		if li.UserOverride.Price != nil {
			ov["price"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(*li.UserOverride.Price, 10)}
		}
		if len(ov) > 0 {
			im["user_override"] = &types.AttributeValueMemberM{Value: ov}
		}
	}
	if li.Removed {
		im["removed"] = &types.AttributeValueMemberBOOL{Value: true}
	}
	if len(li.UnitStates) > 0 {
		unitStates := make([]types.AttributeValue, 0, len(li.UnitStates))
		for _, u := range li.UnitStates {
			um := map[string]types.AttributeValue{
				"status": &types.AttributeValueMemberS{Value: u.Status},
			}
			if u.CancelReason != "" {
				um["cancel_reason"] = &types.AttributeValueMemberS{Value: u.CancelReason}
			}
			if u.CancelledAt != 0 {
				um["cancelled_at"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(u.CancelledAt, 10)}
			}
			unitStates = append(unitStates, &types.AttributeValueMemberM{Value: um})
		}
		im["unit_states"] = &types.AttributeValueMemberL{Value: unitStates}
	}
	if len(li.ParcelUnits) > 0 {
		parcel := make([]types.AttributeValue, 0, len(li.ParcelUnits))
		for _, p := range li.ParcelUnits {
			parcel = append(parcel, &types.AttributeValueMemberBOOL{Value: p})
		}
		im["parcel_units"] = &types.AttributeValueMemberL{Value: parcel}
	}
	return im
}

func decodeLineItem(m map[string]types.AttributeValue) LineItem {
	li := LineItem{}
	if s, ok := m["id"].(*types.AttributeValueMemberS); ok {
		li.ID = s.Value
	}
	if s, ok := m["name"].(*types.AttributeValueMemberS); ok {
		li.Name = s.Value
	}
	if s, ok := m["menu_item_id"].(*types.AttributeValueMemberS); ok {
		li.MenuItemID = s.Value
	}
	if s, ok := m["category"].(*types.AttributeValueMemberS); ok {
		li.Category = s.Value
	}
	if s, ok := m["internal_name"].(*types.AttributeValueMemberS); ok {
		li.InternalName = s.Value
	}
	li.Quantity, _ = atoiAttr(m, "quantity")
	li.Price, _ = numAttr(m, "price")
	if um, ok := m["user_override"].(*types.AttributeValueMemberM); ok {
		ov := &LineItemOverride{}
		hasOverride := false
		if q, ok := um.Value["quantity"].(*types.AttributeValueMemberN); ok {
			if qty, err := strconv.Atoi(q.Value); err == nil {
				ov.Quantity = &qty
				hasOverride = true
			}
		}
		if p, ok := um.Value["price"].(*types.AttributeValueMemberN); ok {
			if price, err := strconv.ParseInt(p.Value, 10, 64); err == nil {
				ov.Price = &price
				hasOverride = true
			}
		}
		if hasOverride {
			li.UserOverride = ov
		}
	}
	if b, ok := m["removed"].(*types.AttributeValueMemberBOOL); ok {
		li.Removed = b.Value
	}
	if s, ok := m["status"].(*types.AttributeValueMemberS); ok {
		li.Status = s.Value
	}
	if l, ok := m["unit_states"].(*types.AttributeValueMemberL); ok {
		for _, e := range l.Value {
			um, ok := e.(*types.AttributeValueMemberM)
			if !ok {
				continue
			}
			u := UnitState{}
			if s, ok := um.Value["status"].(*types.AttributeValueMemberS); ok {
				u.Status = s.Value
			}
			if s, ok := um.Value["cancel_reason"].(*types.AttributeValueMemberS); ok {
				u.CancelReason = s.Value
			}
			u.CancelledAt, _ = numAttr(um.Value, "cancelled_at")
			li.UnitStates = append(li.UnitStates, u)
		}
	}
	if l, ok := m["parcel_units"].(*types.AttributeValueMemberL); ok {
		for _, e := range l.Value {
			if b, ok := e.(*types.AttributeValueMemberBOOL); ok {
				li.ParcelUnits = append(li.ParcelUnits, b.Value)
			}
		}
	}
	return NormalizeLineItem(li)
}

func encodeServiceFlags(f *SessionServiceFlags) map[string]types.AttributeValue {
	if f == nil {
		return nil
	}
	m := map[string]types.AttributeValue{}
	if f.WelcomeDrinkServed {
		m["welcome_drink_served"] = &types.AttributeValueMemberBOOL{Value: true}
	}
	if f.ComplementaryServed {
		m["complementary_served"] = &types.AttributeValueMemberBOOL{Value: true}
	}
	if f.KidMenuEnabled {
		m["kid_menu_enabled"] = &types.AttributeValueMemberBOOL{Value: true}
	}
	if f.KidMenuServed {
		m["kid_menu_served"] = &types.AttributeValueMemberBOOL{Value: true}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

func decodeServiceFlags(item map[string]types.AttributeValue) *SessionServiceFlags {
	m, ok := item["service_flags"].(*types.AttributeValueMemberM)
	if !ok {
		return nil
	}
	f := &SessionServiceFlags{}
	if b, ok := m.Value["welcome_drink_served"].(*types.AttributeValueMemberBOOL); ok {
		f.WelcomeDrinkServed = b.Value
	}
	if b, ok := m.Value["complementary_served"].(*types.AttributeValueMemberBOOL); ok {
		f.ComplementaryServed = b.Value
	}
	if b, ok := m.Value["kid_menu_enabled"].(*types.AttributeValueMemberBOOL); ok {
		f.KidMenuEnabled = b.Value
	}
	if b, ok := m.Value["kid_menu_served"].(*types.AttributeValueMemberBOOL); ok {
		f.KidMenuServed = b.Value
	}
	return f
}
